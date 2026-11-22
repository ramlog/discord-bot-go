package model

import (
	"database/sql"
	"sort"
	"sync"
	"time"

	"github.com/cyb3rplis/discord-bot-go/config"
)

type Model struct {
	Db     *sql.DB
	Config *config.Config
	Mu     *sync.Mutex
}

// New returns a new Model struct
func New(m *Model) *Model {
	cfg := config.GetConfig()
	return &Model{Db: m.Db, Config: cfg}
}

// GuildState holds the runtime state the bot keeps for one Discord server.
// Before multi-server support this lived in package-level globals, which meant
// a stop button or a playing sound on one server affected every server.
type GuildState struct {
	ID   string
	Name string

	playMu      sync.Mutex    // serializes playback within this guild
	stopMu      sync.Mutex    // guards currentStop
	currentStop chan struct{} // closed to stop the active or pending sound

	activityMu   sync.Mutex
	lastActivity time.Time
}

var (
	guildsMu sync.RWMutex
	guilds   = map[string]*GuildState{}
)

// Guild returns the state for a guild, creating it on first use.
func Guild(guildID string) *GuildState {
	guildsMu.RLock()
	g, ok := guilds[guildID]
	guildsMu.RUnlock()
	if ok {
		return g
	}

	guildsMu.Lock()
	defer guildsMu.Unlock()
	if g, ok = guilds[guildID]; !ok {
		g = &GuildState{ID: guildID, lastActivity: time.Now()}
		guilds[guildID] = g
	}
	return g
}

// RegisterGuild creates or updates the state of a guild the bot is in.
func RegisterGuild(guildID, name string) *GuildState {
	g := Guild(guildID)
	guildsMu.Lock()
	g.Name = name
	guildsMu.Unlock()
	return g
}

// RemoveGuild forgets a guild, e.g. after the bot was kicked.
func RemoveGuild(guildID string) {
	guildsMu.Lock()
	defer guildsMu.Unlock()
	if g, ok := guilds[guildID]; ok {
		g.StopPlayback()
		delete(guilds, guildID)
	}
}

// Guilds returns a snapshot of all known guilds, sorted by ID.
func Guilds() []*GuildState {
	guildsMu.RLock()
	defer guildsMu.RUnlock()
	out := make([]*GuildState, 0, len(guilds))
	for _, g := range guilds {
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Touch marks the guild as active (used for the voice inactivity timeout).
func (g *GuildState) Touch() {
	g.activityMu.Lock()
	g.lastActivity = time.Now()
	g.activityMu.Unlock()
}

// LastActivity returns when a sound was last played in this guild.
func (g *GuildState) LastActivity() time.Time {
	g.activityMu.Lock()
	defer g.activityMu.Unlock()
	return g.lastActivity
}

// StartPlayback stops whatever is playing in this guild and registers a new
// playback. The returned channel is closed when this playback should stop;
// done must be called when the playback ends.
func (g *GuildState) StartPlayback() (stop <-chan struct{}, done func()) {
	ch := make(chan struct{})
	g.stopMu.Lock()
	if g.currentStop != nil {
		close(g.currentStop)
	}
	g.currentStop = ch
	g.stopMu.Unlock()

	return ch, func() {
		g.stopMu.Lock()
		if g.currentStop == ch {
			g.currentStop = nil
		}
		g.stopMu.Unlock()
	}
}

// StopPlayback signals the active sound of this guild to stop without blocking.
func (g *GuildState) StopPlayback() {
	g.stopMu.Lock()
	defer g.stopMu.Unlock()
	if g.currentStop != nil {
		close(g.currentStop)
		g.currentStop = nil
	}
}

// Playing reports whether a sound is currently playing (or queued) in this guild.
func (g *GuildState) Playing() bool {
	g.stopMu.Lock()
	defer g.stopMu.Unlock()
	return g.currentStop != nil
}

// LockPlayer / UnlockPlayer ensure only one sound plays per guild at a time.
func (g *GuildState) LockPlayer()   { g.playMu.Lock() }
func (g *GuildState) UnlockPlayer() { g.playMu.Unlock() }
