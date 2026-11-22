package view

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/cyb3rplis/discord-bot-go/logger"
	"github.com/cyb3rplis/discord-bot-go/model"
)

const (
	maxInteractions = 15               // interactions allowed per window...
	resetDuration   = 15 * time.Second // ...before the user is sent to the gulag
	spamGulagMins   = 1
)

var (
	rateMu          sync.Mutex
	userLastAction  = make(map[string]time.Time)
	userActionCount = make(map[string]int)
)

// AllowInteraction is the spam protection for ALL slash commands and button
// clicks (autocomplete excluded, Discord throttles that itself). It returns
// false and answers the interaction if the user exceeded the limit.
func (a *API) AllowInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if i.Type != discordgo.InteractionApplicationCommand && i.Type != discordgo.InteractionMessageComponent {
		return true
	}
	user := i.Member.User

	rateMu.Lock()
	last, seen := userLastAction[user.ID]
	if seen && time.Since(last) < resetDuration {
		userActionCount[user.ID]++
	} else {
		userActionCount[user.ID] = 1
	}
	userLastAction[user.ID] = time.Now()
	count := userActionCount[user.ID]
	rateMu.Unlock()

	if count <= maxInteractions {
		return true
	}

	// only gulag + log once when crossing the limit, afterwards just drop silently-ish
	if count == maxInteractions+1 {
		log.WarningLog.Printf("Guild %s: user %s (%s) exceeded the rate limit, gulag for %d minute(s)",
			i.GuildID, model.DisplayName(user), user.ID, spamGulagMins)
		if err := a.model.GulagUser(user, spamGulagMins); err != nil {
			log.ErrorLog.Println("error gulagging user:", err)
		}
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🧱  Slow down! You are in the Gulag for %d minute(s).", spamGulagMins),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	return false
}

// PruneRateLimits removes old entries so the maps don't grow forever.
func PruneRateLimits() {
	rateMu.Lock()
	defer rateMu.Unlock()
	for id, t := range userLastAction {
		if time.Since(t) > resetDuration {
			delete(userLastAction, id)
			delete(userActionCount, id)
		}
	}
}

// Safe runs fn and turns a panic into a log entry instead of crashing the whole
// bot (which would take it offline on every server).
func Safe(what string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorLog.Printf("recovered from panic in %s: %v\n%s", what, r, debug.Stack())
		}
	}()
	fn()
}
