package view

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/cyb3rplis/discord-bot-go/config"
	"github.com/cyb3rplis/discord-bot-go/model"

	log "github.com/cyb3rplis/discord-bot-go/logger"
)

// requireOwner answers with an error and returns false if the interaction user
// is not a configured bot owner. Used for commands that affect ALL servers.
func (a *API) requireOwner(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if a.model.Config.IsOwner(i.Member.User.ID) {
		return true
	}
	log.WarningLog.Printf("Guild %s: user %s (%s) tried to use /%s without being a bot owner",
		i.GuildID, model.DisplayName(i.Member.User), i.Member.User.ID, i.ApplicationCommandData().Name)
	_ = a.SendInteractionRespond("⛔  This command changes the bot on all servers and is reserved for the bot owners.", s, i)
	return false
}

// resolvedUser returns the full user object of a USER-type command option.
func resolvedUser(i *discordgo.InteractionCreate, opt *discordgo.ApplicationCommandInteractionDataOption) *discordgo.User {
	id := opt.Value.(string)
	if res := i.ApplicationCommandData().Resolved; res != nil {
		if u, ok := res.Users[id]; ok {
			return u
		}
	}
	return &discordgo.User{ID: id}
}

func (a *API) PromptInteractionGulag(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand || i.ApplicationCommandData().Name != "gulag" {
		return
	}
	if !a.requireOwner(s, i) {
		return
	}
	interactionUser := i.Member.User
	option := i.ApplicationCommandData().Options[0]

	switch option.Name {
	case "list":
		users, err := a.model.GetUsers()
		if err != nil {
			log.ErrorLog.Println("error getting users:", err)
			_ = a.SendInteractionRespond("🧱  Could not load the gulag list", s, i)
			return
		}
		message := "🧱  Gulagged Users (all servers):\n"
		count := 0
		for _, u := range users {
			if u, ok := SetUserGulagRemaining(u); ok {
				message += fmt.Sprintf("> » User: %s - Remaining: %s\n", u.User.GlobalName, u.Remaining)
				count++
			}
		}
		if count == 0 {
			message = "🧱  No users are gulagged\n"
		}
		_ = a.SendInteractionRespond(message, s, i)

	case "add":
		if len(option.Options) < 2 {
			return
		}
		target := resolvedUser(i, option.Options[0])
		minutes := int(option.Options[1].IntValue())
		if minutes <= 0 {
			_ = a.SendInteractionRespond(fmt.Sprintf("🧱  Invalid timeout value: %d\n", minutes), s, i)
			return
		}
		if err := a.model.GulagUser(target, minutes); err != nil {
			log.ErrorLog.Println("error putting user into gulag:", err)
			_ = a.SendInteractionRespond(fmt.Sprintf("🧱  error putting user into gulag: %s --> %v \n", model.DisplayName(target), err), s, i)
			return
		}
		log.InfoLog.Printf("Owner %s put user %s (%s) in the gulag for %d minutes", model.DisplayName(interactionUser), model.DisplayName(target), target.ID, minutes)
		_ = a.SendInteractionRespond(fmt.Sprintf("🧱  %s has been put into the gulag for %d minutes (all servers)\n", target.Mention(), minutes), s, i)

	case "remove":
		if len(option.Options) < 1 {
			return
		}
		target := resolvedUser(i, option.Options[0])
		if err := a.model.ReleaseUser(target); err != nil {
			log.ErrorLog.Println("error releasing user from gulag:", err)
			_ = a.SendInteractionRespond(fmt.Sprintf("🧱  error releasing %s from gulag --> %v \n", model.DisplayName(target), err), s, i)
			return
		}
		log.InfoLog.Printf("Owner %s released %s (%s) from gulag", model.DisplayName(interactionUser), model.DisplayName(target), target.ID)
		_ = a.SendInteractionRespond(fmt.Sprintf("🧱  %s has been released from the gulag\n", target.Mention()), s, i)

	default:
		_ = a.SendInteractionRespond("🧱  Something went wrong...", s, i)
	}
}

func SetUserGulagRemaining(user config.ExtendedUser) (config.ExtendedUser, bool) {
	if user.Gulagged.Valid {
		rem := time.Until(user.Gulagged.Time)
		if rem > 0 {
			user.Remaining = rem.Truncate(time.Second)
			return user, true
		}
	}
	return user, false
}
