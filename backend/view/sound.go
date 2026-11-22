package view

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	log "github.com/cyb3rplis/discord-bot-go/logger"
	"github.com/cyb3rplis/discord-bot-go/model"

	"github.com/bwmarrin/discordgo"
)

const opusSendTimeout = 2 * time.Second // max wait for the voice connection to accept a frame

// tempAudioName is the per-guild file name used by /audio, so two servers
// downloading audio at the same time don't overwrite each other's file.
func (a *API) tempAudioName(guildID string) string {
	return a.model.Config.AudioTemp + "_" + guildID
}

type Entry struct {
	ID   int
	Name string
}

// PromptInteractionPlaySound handles /play and /random.
func (a *API) PromptInteractionPlaySound(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}
	data := i.ApplicationCommandData()
	if data.Name != "play" && data.Name != "random" {
		return
	}

	_ = a.SendInteractionRespond("➡ Playing sound", s, i)

	vs, err := a.VoiceChannelCheck(s, i)
	if err != nil {
		return
	}
	if a.rejectIfGulagged(s, i) {
		return
	}

	var req playRequest
	switch data.Name {
	case "play":
		soundName := data.Options[0].StringValue()
		// /play accepts free text (autocomplete is only a suggestion), so make
		// sure the sound exists before its name is posted in the channel
		if sound, err := a.model.GetSound(soundName); err != nil || sound.Name == "" {
			_ = a.UpdateInteractionResponse("🎶  Sound not found: "+escapeMarkdown(soundName), s, i)
			return
		}
		req = playRequest{soundName: soundName, display: soundName, countStats: true}

	case "random":
		category := ""
		if len(data.Options) > 0 {
			category = data.Options[0].StringValue()
		}
		sound, err := a.model.GetRandomSound(category)
		if err != nil {
			msg := "🎲  No sounds found"
			if category != "" {
				msg += " in category " + escapeMarkdown(category)
			}
			_ = a.UpdateInteractionResponse(msg, s, i)
			return
		}
		req = playRequest{soundName: sound.Name, display: "🎲 " + sound.Name, countStats: true}
		_ = a.UpdateInteractionResponse("🎲  Playing random sound: "+escapeMarkdown(sound.Name), s, i)
	}

	if err := a.announceAndPlay(s, i, vs.ChannelID, req); err != nil {
		log.ErrorLog.Println("error playing sound:", err)
		_ = a.UpdateInteractionResponse("➡ Could not play the sound", s, i)
	}
}

// PlaySound plays a sound in the given voice channel of a guild.
// isTemp=true plays the guild's last /audio download from disk instead of the DB.
func (a *API) PlaySound(s *discordgo.Session, guildID, channelID, soundName string, isTemp bool) error {
	g := model.Guild(guildID)

	// Stop the current sound of THIS guild and register this one as the latest request.
	playback, done := g.StartPlayback()
	defer done()
	stop := playback.Stopped()

	g.LockPlayer()
	defer g.UnlockPlayer()
	g.Touch()

	// A newer request or the stop button superseded this one while waiting.
	select {
	case <-stop:
		return nil
	default:
	}

	// Load the sound file.
	var err error
	var buffer [][]byte
	if isTemp {
		buffer, err = a.model.LoadSoundFS(filepath.Join(a.model.Config.DataDir, soundName+".dca")) //play file from system if function (audio) is played
	} else {
		buffer, err = a.model.LoadSound(soundName)
	}
	if err != nil {
		log.ErrorLog.Printf("error loading sound %s, %v ", soundName, err)
		return err
	}

	// Join the provided voice channel.
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return err
	}

	// Sleep for a specified amount of time before playing the sound
	time.Sleep(150 * time.Millisecond)

	// Start speaking.
	_ = vc.Speaking(true)

	// Send the buffer data, then once more for every requested repeat.
	// Closing `stop` (stop button / newer sound) aborts the sound AND all repeats.
	for {
		for _, buff := range buffer {
			select {
			case vc.OpusSend <- buff:
			case <-stop:
				_ = vc.Speaking(false)
				return nil
			case <-time.After(opusSendTimeout):
				// Voice sender is dead, drop the connection so the next play rejoins cleanly.
				log.ErrorLog.Printf("voice connection stalled while playing %s, disconnecting", soundName)
				_ = vc.Speaking(false)
				if err := vc.Disconnect(); err != nil {
					log.ErrorLog.Println("error disconnecting stalled voice connection:", err)
				}
				return fmt.Errorf("voice connection stalled")
			}
		}
		if !playback.TakeRepeat() {
			break
		}
		// short gap between repetitions, still interruptible
		select {
		case <-stop:
			_ = vc.Speaking(false)
			return nil
		case <-time.After(300 * time.Millisecond):
		}
	}

	// Stop speaking
	_ = vc.Speaking(false)

	// Sleep for a specified amount of time before ending.
	time.Sleep(250 * time.Millisecond)

	// Update the bot activity of this guild
	g.Touch()
	return nil
}

// PlayAudio plays the guild's last downloaded /audio file.
func (a *API) PlayAudio(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	audioName := a.tempAudioName(i.GuildID)
	if _, err := os.Stat(filepath.Join(a.model.Config.DataDir, audioName+".dca")); err != nil {
		_ = a.UpdateInteractionResponse("🎶  No audio has been played on this server yet", s, i)
		return err
	}

	vs, err := a.VoiceChannelCheck(s, i)
	if err != nil {
		return err
	}
	_ = a.UpdateInteractionResponse("🎶  Playing audio", s, i)

	return a.announceAndPlay(s, i, vs.ChannelID, playRequest{
		soundName: audioName, display: "audio", isTemp: true,
	})
}

// SyncDatabaseWithFileSystem synchronizes the database with the filesystem.
func (a *API) SyncDatabaseWithFileSystem(folderMap map[string][]string) error {
	existingCategories, _ := a.model.GetCategoriesM()

	var wg sync.WaitGroup                                        // WaitGroup to wait for all Goroutines to finish
	soundProcessingChan := make(chan struct{}, runtime.NumCPU()) // limit parallelism to maximum available threads Goroutines

	for folder, files := range folderMap {
		categoryID, err := a.getOrCreateCategoryID(folder, existingCategories)
		if err != nil {
			log.WarningLog.Printf("Failed to process category %s: %v", folder, err)
			continue
		}

		for _, file := range files {
			wg.Add(1)                         // increment the WaitGroup counter
			soundProcessingChan <- struct{}{} // acquire a token to limit concurrency

			go func(folder, file string, categoryID int) {
				defer wg.Done()
				defer func() { <-soundProcessingChan }()

				if err := a.processSoundFile(folder, file, categoryID); err != nil {
					log.WarningLog.Printf("failed to process file %s in folder %s: %v", file, folder, err)
				}
			}(folder, file, categoryID)
		}
	}

	wg.Wait()                                          // wait for all Goroutines to finish
	a.cleanUpCategories(folderMap, existingCategories) // remove orphaned entries from the database

	return nil
}

// getOrCreateCategoryID retrieves or creates a category and returns its ID.
func (a *API) getOrCreateCategoryID(folder string, existingCategories map[string]int) (int, error) {
	if categoryID, exists := existingCategories[folder]; exists {
		return categoryID, nil
	}
	if err := a.model.AddCategory(folder); err != nil {
		if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return 0, err
		}
	}
	return a.model.GetCategoryByID(folder)
}

// processSoundFile handles the conversion and addition of a sound file.
func (a *API) processSoundFile(folder, file string, categoryID int) error {
	soundPathMP3 := filepath.Join(a.model.Config.SoundsDir, folder, file+".mp3")
	soundPathDCA := filepath.Join(a.model.Config.SoundsDir, folder, file+".dca")

	if _, err := os.Stat(soundPathMP3); os.IsNotExist(err) {
		return fmt.Errorf("MP3 file not found: %s", soundPathMP3)
	}

	fileHash, err := model.ComputeFileHash(soundPathMP3)
	if err != nil {
		return fmt.Errorf("failed to compute hash: %v", err)
	}

	if err := a.ConvertMP3ToDCA(file, folder); err != nil {
		return fmt.Errorf("error converting mp3 to dca: %v", err)
	}

	soundBytes, err := os.ReadFile(soundPathDCA)
	if err != nil {
		return fmt.Errorf("failed to read DCA file: %v", err)
	}

	if err := a.model.AddSound(categoryID, file, fileHash, soundBytes); err != nil {
		return fmt.Errorf("failed to add sound: %v", err)
	}

	err = os.Remove(soundPathDCA)
	if err != nil {
		return fmt.Errorf("failed to remove dca file after insert into db: %v", err)
	}

	err = os.Rename(soundPathMP3, soundPathMP3+".done")
	if err != nil {
		return fmt.Errorf("failed to rename mp3 file after insert into db: %v", err)
	}

	return nil
}

// cleanUpCategories removes database categories (subfolders) that no longer exist in the filesystem.
func (a *API) cleanUpCategories(folderMap map[string][]string, existingCategories map[string]int) {
	for folder, categoryID := range existingCategories {
		if _, exists := folderMap[folder]; !exists {
			if err := a.model.RemoveCategory(categoryID); err == nil {
				log.InfoLog.Printf("Removed category %s (ID: %d)", folder, categoryID)
			}
			continue
		}
	}
}

func (a *API) VoiceChannelCheck(s *discordgo.Session, i *discordgo.InteractionCreate) (*discordgo.VoiceState, error) {
	userInVS := false
	var voiceState *discordgo.VoiceState
	// Find the guild the interaction came from
	guild, err := s.State.Guild(i.GuildID)
	if err != nil {
		log.ErrorLog.Println("error finding guild:", err)
		return voiceState, err
	}

	user := i.Member.User

	for _, vs := range guild.VoiceStates {
		if vs.UserID == user.ID {
			userInVS = true
			voiceState = vs
		}
	}
	if !userInVS {
		// If the user is not in a voice channel, send an error message and avoid processing the audio
		log.InfoLog.Printf("User %s tried to play sound but is not in a voice channel", user.GlobalName)
		msg := "You need to be in a voice channel to play sounds " + user.Mention()
		_, err = a.SendMessage(msg, s, i, false)
		if err != nil {
			log.ErrorLog.Println("error[sound5] sending message:", err)
		}
		return voiceState, fmt.Errorf("user not in voice channel, quitting early to avoid delay")
	}

	return voiceState, nil
}
