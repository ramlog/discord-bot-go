package view

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	log "github.com/cyb3rplis/discord-bot-go/logger"
)

// PromptInteractionHelp answers /help with an overview that only lists what
// the requesting user is actually allowed to use. Only they can see it.
func (a *API) PromptInteractionHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand || i.ApplicationCommandData().Name != "help" {
		return
	}
	userID := i.Member.User.ID
	msg := helpMessage(a.model.Config.IsMod(userID), a.model.Config.IsOwner(userID))
	if err := a.SendInteractionRespond(msg, s, i); err != nil {
		log.ErrorLog.Println("help interaction", err)
	}
}

// helpMessage builds the /help text. Discord limits messages to 2000 characters.
func helpMessage(isMod, isOwner bool) string {
	var b strings.Builder

	b.WriteString("## 🔊 Soundboard Help\n")
	b.WriteString("**Sounds**\n")
	b.WriteString("> `/buttons list` – show the soundboard (category buttons + 🎲)\n")
	b.WriteString("> `/play sound` – play a sound by name\n")
	b.WriteString("> `/random [category]` – play a random sound\n")
	b.WriteString("**Buttons**\n")
	b.WriteString("> **Category** – show its sounds · **Sound** – play it · **🎲** – random sound\n")
	b.WriteString("> **Stop Sound** – stop the sound and all repeats\n")
	b.WriteString("> **🔁 Repeat** – play the current sound again (max. 3×)\n")
	b.WriteString("**Favorites** (shared across all servers)\n")
	b.WriteString("> `/favorite add|remove sound` · `/favorite buttons`\n")
	b.WriteString("**Statistics** (this server)\n")
	b.WriteString("> `/stats sounds` · `/stats users` · `/stats me`\n")
	b.WriteString("**Misc**\n")
	b.WriteString("> `/misc leave` – kick the bot from voice · `/misc version` – bot version\n")

	if isMod {
		b.WriteString("\n**🛠️ Moderators**\n")
		b.WriteString("> `/audio play url` – play audio from YouTube & co.\n")
		b.WriteString("> `/audio last` – replay the last audio\n")
	}
	if isOwner {
		b.WriteString("\n**👑 Owners** (affects ALL servers)\n")
		b.WriteString("> `/manage create url name category [start_time] [end_time]` – new sound\n")
		b.WriteString("> `/manage delete name` · `/manage move name category`\n")
		b.WriteString("> `/gulag add user minutes` · `/gulag remove user` · `/gulag list`\n")
	}

	b.WriteString("\n-# Spam protection: more than 15 actions in 15 s = 1 minute gulag")
	return b.String()
}
