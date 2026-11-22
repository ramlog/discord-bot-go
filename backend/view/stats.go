package view

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	log "github.com/cyb3rplis/discord-bot-go/logger"
	"github.com/cyb3rplis/discord-bot-go/model"
)

func (a *API) PromptInteractionStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	//get userID
	interactionUser := i.Member.User

	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case "stats":
			option := i.ApplicationCommandData().Options[0]
			switch option.Name {
			case "sounds":
				err := a.SendInteractionRespond("👉 Getting sound statistics", s, i)
				if err != nil {
					log.ErrorLog.Println("error executing stats command:", err)
				}
				stats, err := a.getStatsSounds(i.GuildID)
				if err != nil {
					log.ErrorLog.Println("error executing stats command:", err)
				}
				err = a.UpdateInteractionResponse(stats, s, i)
				if err != nil {
					log.ErrorLog.Println("error executing stats command:", err)
				}
			case "users":
				err := a.SendInteractionRespond("👉 Getting user statistics", s, i)
				if err != nil {
					log.ErrorLog.Println("error executing stats command:", err)
				}
				stats, err := a.getStatsUsers(i.GuildID)
				if err != nil {
					log.ErrorLog.Println("error executing stats command:", err)
				}
				err = a.UpdateInteractionResponse(stats, s, i)
				if err != nil {
					log.ErrorLog.Println("error executing stats command:", err)
				}
			case "me":
				err := a.SendInteractionRespond("👉 Getting your statistics", s, i)
				if err != nil {
					log.ErrorLog.Println("error executing stats command:", err)
				}
				stats, err := a.getStatsMe(i.GuildID, interactionUser)
				if err != nil {
					log.ErrorLog.Println("error executing stats command:", err)
				}
				err = a.UpdateInteractionResponse(stats, s, i)
				if err != nil {
					log.ErrorLog.Println("error executing stats command:", err)
				}
			default:
				err := a.SendInteractionRespond("👉  Something went wrong...", s, i)
				if err != nil {
					log.ErrorLog.Println("fallback to default stats handler", err)
				}
			}
		}
	}
}

func (a *API) getStatsSounds(guildID string) (string, error) {
	soundStats, err := a.model.GetSoundStatistics(guildID)
	if err != nil {
		log.ErrorLog.Printf("error getting sound statistics: %v", err)
	}
	sortedKeys := model.SortMapKeysByValue(soundStats)
	content := strings.Builder{}
	content.Write([]byte("🔥  Top 10 sounds on this server: \n\n"))
	for _, c := range sortedKeys {
		content.Write([]byte(fmt.Sprintf("> %dx:\t%s\n", soundStats[c], c)))
	}
	return content.String(), nil
}

func (a *API) getStatsUsers(guildID string) (string, error) {
	userStats, err := a.model.GetAllUserStatistics(guildID)
	if err != nil {
		log.ErrorLog.Printf("error getting all users statistics: %v", err)
	}
	sortedKeys := model.SortMapKeysByValue(userStats)

	content := strings.Builder{}
	content.Write([]byte("🔥  Top 10 users on this server: \n\n"))
	for _, c := range sortedKeys {
		content.Write([]byte(fmt.Sprintf("> %dx:\t%s\n", userStats[c], c)))
	}
	return content.String(), nil
}

func (a *API) getStatsMe(guildID string, user *discordgo.User) (string, error) {
	userStats, err := a.model.GetUserStatistics(guildID, user, 10)
	if err != nil {
		log.ErrorLog.Printf("error getting user statistics: %v", err)
	}
	content := strings.Builder{}
	content.Write([]byte("🔥  " + user.Mention() + "'s top 10 played sounds on this server: \n\n"))
	for _, s := range userStats {
		content.Write([]byte(fmt.Sprintf("> %dx:\t%s\n", s.Count, s.Name)))
	}
	return content.String(), nil
}
