package view

import (
	"fmt"
	"runtime"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/cyb3rplis/discord-bot-go/model"
	"github.com/cyb3rplis/discord-bot-go/version"

	log "github.com/cyb3rplis/discord-bot-go/logger"
)

func (a *API) PromptInteractionMisc(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type == discordgo.InteractionApplicationCommand {
		switch i.ApplicationCommandData().Name {
		case "misc":
			option := i.ApplicationCommandData().Options[0]
			switch option.Name {
			case "leave":
				err := a.SendInteractionRespond("👋  Leaving Voice Channel", s, i)
				if err != nil {
					log.ErrorLog.Println("leave interaction", err)
				}

				// Get the guild ID
				guildID := i.GuildID
				if guildID == "" {
					err := a.UpdateInteractionResponse("👋  Something went wrong...", s, i)

					if err != nil {
						log.ErrorLog.Println("leave interaction", err)
					}
					return
				}

				voiceConnection, ok := s.VoiceConnections[guildID]
				if !ok {
					log.ErrorLog.Println("Bot is not connected to a voice channel in this guild")
					return
				}

				// Leave the voice channel
				err = voiceConnection.Disconnect()
				if err != nil {
					log.ErrorLog.Printf("Error disconnecting from the voice channel: %v\n", err)
				} else {
					err := a.UpdateInteractionResponse("👋  Bye Bye", s, i)

					if err != nil {
						log.ErrorLog.Println("leave interaction", err)
					}
				}

			case "version":
				if err := a.SendInteractionRespond(versionMessage(), s, i); err != nil {
					log.ErrorLog.Println("version interaction", err)
				}

			default:
				err := a.SendInteractionRespond("👋  Something went wrong...", s, i)
				if err != nil {
					log.ErrorLog.Println("fallback to default misc handler", err)
				}
			}
		}
	}
}

// versionMessage builds the /misc version answer.
func versionMessage() string {
	commit := "`" + version.ShortCommit() + "`"
	if version.RepoURL != "" && version.Commit != "unknown" {
		commit = fmt.Sprintf("[`%s`](<%s/commit/%s>)", version.ShortCommit(), version.RepoURL, version.Commit)
	}
	uptime := time.Since(version.StartedAt).Truncate(time.Second)
	return fmt.Sprintf("🤖  **discord-bot-go**\n"+
		"> Version: **%s**\n"+
		"> Commit: %s\n"+
		"> Built: %s\n"+
		"> Go: %s\n"+
		"> Uptime: %s\n"+
		"> Servers: %d",
		version.Version, commit, version.BuildDate, runtime.Version(), uptime, len(model.Guilds()))
}
