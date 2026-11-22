package view

import (
	"database/sql"
	"fmt"
	"strings"

	log "github.com/cyb3rplis/discord-bot-go/logger"
	"github.com/cyb3rplis/discord-bot-go/model"

	"github.com/bwmarrin/discordgo"
)

var BotReady = false

// InteractionHandler handles interaction events (e.g., button clicks)
func (a *API) InteractionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}
	customID := i.MessageComponentData().CustomID

	// Acknowledge the interaction
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.ErrorLog.Println("Failed to respond to interaction:", err)
		return
	}

	// Check if the user is in the Gulag
	user, err := a.model.SetUserGulaggedValue(i.Member.User)
	if err != nil && err != sql.ErrNoRows {
		log.ErrorLog.Println("error getting user from username:", err)
		return
	} else {
		if user, ok := SetUserGulagRemaining(user); ok {
			msg := fmt.Sprintf(user.User.Mention()+" you are in the Gulag for another %s", user.Remaining)
			_, err = a.SendMessage(msg, s, i, true)
			if err != nil {
				log.ErrorLog.Printf("error[int1] sending message: %v", err)
			}
			return
		}
	}

	switch {
	case strings.HasPrefix(customID, "play_sound_"):
		a.handlePlaySoundInteraction(s, i)
	case strings.HasPrefix(customID, "list_sounds_"):
		a.handleListSoundsInteraction(s, i)
	case customID == stopSoundID:
		a.handleStopSoundInteraction(s, i)
	case customID == repeatSoundID:
		a.handleRepeatInteraction(s, i)
	case customID == randomSoundID:
		a.handleRandomSoundInteraction(s, i)
	default:
		log.ErrorLog.Println("unknown interaction:", customID)
	}
}

// handleStopSoundInteraction handles the stop sound interaction
func (a *API) handleStopSoundInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// only stops the sound on the server the button was clicked on
	model.Guild(i.GuildID).StopPlayback()

	// Delete the message that contains the stop button
	if i.Message != nil {
		if err := s.ChannelMessageDelete(i.ChannelID, i.Message.ID); err != nil {
			log.ErrorLog.Println("error deleting stop sound button:", err)
		}
	}
}

func (a *API) handlePlaySoundInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.SplitN(strings.TrimPrefix(customID, "play_sound_"), "_", 2)
	if len(parts) != 2 {
		log.ErrorLog.Println("Invalid custom ID format")
		return
	}
	soundName := parts[1]

	vs, err := a.VoiceChannelCheck(s, i)
	if err != nil {
		return
	}
	if err := a.announceAndPlay(s, i, vs.ChannelID, playRequest{
		soundName: soundName, display: soundName, countStats: true,
	}); err != nil {
		log.ErrorLog.Println("error playing sound:", err)
	}
}

// HandleListSoundsInteraction handles the list sounds interaction
func (a *API) handleListSoundsInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	interactionUser := i.Member.User
	log.InfoLog.Println("Handling list sounds interaction")
	// Extract the category from the custom ID
	customID := ""
	if i.Interaction.Type != 3 {
		log.ErrorLog.Println("Invalid interaction type")
		return
	}

	customID = i.Interaction.MessageComponentData().CustomID

	category := strings.TrimPrefix(customID, "list_sounds_")
	msg := "➡ Sounds in category - " + category
	_, err := a.SendMessage(msg, s, i, false)
	if err != nil {
		log.ErrorLog.Println("error[int4] sending message:", err)
	}

	// Get all sound files in the subfolder
	sounds, err := a.model.GetSounds(category)
	if err != nil {
		log.ErrorLog.Println("error listing sounds in subfolder:", err)
	}
	if len(sounds) == 0 {
		msg := "No sounds found in this category."
		_, err = a.SendMessage(msg, s, i, false)
		if err != nil {
			log.ErrorLog.Println("error[int5] sending message:", err)
		}
		return
	}

	//build buttons for each sound
	buttons := model.BuildSoundButtons(sounds, category, discordgo.SecondaryButton)
	//build messages
	messages := model.BuildMessages(buttons, nil)

	log.InfoLog.Printf("User: %s listed sounds in category: %s", interactionUser.GlobalName, category)
	for _, msg := range messages {
		_, err = a.SendMessageComplex(msg, s, i, false)
		if err != nil {
			log.ErrorLog.Println("error[int6] sending message:", err)
		}
	}
}
