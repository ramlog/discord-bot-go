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

	playMu  sync.Mutex // serializes playback within this guild
	stopMu  sync.Mutex // guards current
	current *Playback  // the active (or pending) playback, nil if silent

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

// Playback is one requested sound including its queued repeats.
type Playback struct {
	stop    chan struct{} // closed to abort the sound and all its repeats
	mu      sync.Mutex
	pending int // repeats still to play
	used    int // repeats requested so far (for the limit)
}

// Stopped is closed when the playback should end (stop button or newer sound).
func (p *Playback) Stopped() <-chan struct{} { return p.stop }

// TakeRepeat consumes one queued repeat; false if none is left.
func (p *Playback) TakeRepeat() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pending == 0 {
		return false
	}
	p.pending--
	return true
}

// StartPlayback stops whatever is playing in this guild and registers a new
// playback. done must be called when the playback ends.
func (g *GuildState) StartPlayback() (p *Playback, done func()) {
	p = &Playback{stop: make(chan struct{})}
	g.stopMu.Lock()
	if g.current != nil {
		close(g.current.stop)
	}
	g.current = p
	g.stopMu.Unlock()

	return p, func() {
		g.stopMu.Lock()
		if g.current == p {
			g.current = nil
		}
		g.stopMu.Unlock()
	}
}

// StopPlayback aborts the active sound of this guild AND all its queued repeats.
func (g *GuildState) StopPlayback() {
	g.stopMu.Lock()
	defer g.stopMu.Unlock()
	if g.current != nil {
		close(g.current.stop)
		g.current = nil
	}
}

// AddRepeat queues one more repetition of the active sound, up to max per
// playback. Returns how many repeats were requested in total so far.
func (g *GuildState) AddRepeat(max int) (used int, ok bool) {
	g.stopMu.Lock()
	p := g.current
	g.stopMu.Unlock()
	if p == nil {
		return 0, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.used >= max {
		return p.used, false
	}
	p.used++
	p.pending++
	return p.used, true
}

// Playing reports whether a sound is currently playing (or queued) in this guild.
func (g *GuildState) Playing() bool {
	g.stopMu.Lock()
	defer g.stopMu.Unlock()
	return g.current != nil
}

// LockPlayer / UnlockPlayer ensure only one sound plays per guild at a time.
func (g *GuildState) LockPlayer()   { g.playMu.Lock() }
func (g *GuildState) UnlockPlayer() { g.playMu.Unlock() }
