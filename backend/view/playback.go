package view

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/cyb3rplis/discord-bot-go/logger"
	"github.com/cyb3rplis/discord-bot-go/model"
)

// maxRepeats is how often the repeat button can replay one sound.
const maxRepeats = 3

// Button custom IDs of the "now playing" message and the random button.
const (
	stopSoundID   = "stop_sound"
	repeatSoundID = "repeat_sound"
	randomSoundID = "random_sound"
)

// playRequest describes one sound to announce and play.
type playRequest struct {
	soundName  string // DB sound name, or the temp file name for /audio
	display    string // what the "Currently Playing" message shows
	isTemp     bool   // play the /audio download from disk instead of the DB
	countStats bool   // count this play in the guild statistics
}

// nowPlayingComponents builds the Stop + Repeat buttons. used is how many
// repeats were already requested; the button is disabled at the limit.
func nowPlayingComponents(used int) []discordgo.MessageComponent {
	label := "🔁 Repeat"
	if used > 0 {
		label = fmt.Sprintf("🔁 Repeat (%d/%d)", used, maxRepeats)
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "Stop Sound", Style: discordgo.DangerButton, CustomID: stopSoundID},
			discordgo.Button{Label: label, Style: discordgo.SecondaryButton, CustomID: repeatSoundID, Disabled: used >= maxRepeats},
		}},
	}
}

// randomButtonComponents is the "🎲 Random Sound" button shown under the soundboard.
func randomButtonComponents() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "🎲 Random Sound", Style: discordgo.SuccessButton, CustomID: randomSoundID},
		}},
	}
}

// rejectIfGulagged tells a gulagged user off and returns true.
func (a *API) rejectIfGulagged(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	user, err := a.model.SetUserGulaggedValue(i.Member.User)
	if err != nil && err != sql.ErrNoRows {
		log.ErrorLog.Println("error loading gulag state:", err)
		return false
	}
	if user, ok := SetUserGulagRemaining(user); ok {
		msg := fmt.Sprintf("%s you are in the Gulag for another %s", user.User.Mention(), user.Remaining)
		if _, err := a.SendMessage(msg, s, i, true); err != nil {
			log.ErrorLog.Printf("error sending gulag message: %v", err)
		}
		return true
	}
	return false
}

// announceAndPlay posts the "Currently Playing" message with Stop/Repeat
// buttons, plays the sound (plus requested repeats) and removes the message
// afterwards. Voice and gulag checks must be done by the caller.
func (a *API) announceAndPlay(s *discordgo.Session, i *discordgo.InteractionCreate, voiceChannelID string, req playRequest) error {
	user := i.Member.User

	if req.countStats {
		if err := a.model.AddUser(user); err != nil {
			log.ErrorLog.Println("error adding user:", err)
		}
		if err := a.model.AddUserStatistics(i.GuildID, user, req.soundName); err != nil {
			log.ErrorLog.Println("error adding user statistics:", err)
		}
	}

	st, err := a.SendMessageComplex(&discordgo.MessageSend{
		Content:    "➡ Currently Playing by " + user.Mention() + ": " + req.display,
		Components: nowPlayingComponents(0),
	}, s, i, false)
	if err != nil {
		return err
	}
	if err := a.DeleteOldStopSoundButtons(s, st); err != nil {
		log.ErrorLog.Println("error deleting old stop sound buttons:", err)
	}

	log.InfoLog.Printf("Guild %s: user %s played: %s", i.GuildID, model.DisplayName(user), req.display)
	playErr := a.PlaySound(s, i.GuildID, voiceChannelID, req.soundName, req.isTemp)

	// remove the now-playing message (it may already be gone if Stop was pressed)
	time.Sleep(250 * time.Millisecond)
	_ = s.ChannelMessageDelete(st.ChannelID, st.ID)
	return playErr
}

// handleRepeatInteraction queues one more repetition of the current sound.
func (a *API) handleRepeatInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	used, ok := model.Guild(i.GuildID).AddRepeat(maxRepeats)
	if !ok {
		return // nothing playing anymore or limit reached
	}
	log.InfoLog.Printf("Guild %s: user %s requested repeat %d/%d", i.GuildID, model.DisplayName(i.Member.User), used, maxRepeats)

	// the interaction was acknowledged with a deferred update, so editing the
	// response edits the message the button belongs to
	components := nowPlayingComponents(used)
	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Components: &components}); err != nil {
		log.ErrorLog.Println("error updating repeat button:", err)
	}
}

// handleRandomSoundInteraction plays a random sound from the 🎲 button.
func (a *API) handleRandomSoundInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	vs, err := a.VoiceChannelCheck(s, i)
	if err != nil {
		return
	}
	sound, err := a.model.GetRandomSound("")
	if err != nil {
		log.ErrorLog.Println("error picking random sound:", err)
		return
	}
	err = a.announceAndPlay(s, i, vs.ChannelID, playRequest{
		soundName: sound.Name, display: "🎲 " + sound.Name, countStats: true,
	})
	if err != nil {
		log.ErrorLog.Println("error playing random sound:", err)
	}
}
