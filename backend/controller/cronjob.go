package controller

import (
	"context"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/cyb3rplis/discord-bot-go/logger"
	"github.com/cyb3rplis/discord-bot-go/model"
	"github.com/go-co-op/gocron"
)

func (c *Controller) SyncCronjob(ctx context.Context) {
	interval := time.Minute
	for {
		select {
		case <-time.After(interval):

		case <-ctx.Done():
			return
		}
	}
}

func (c *Controller) SyncSoundDirectories() {
	interval := time.Minute * 1
	s := gocron.NewScheduler(time.UTC)
	s.Every(interval).Do(func() {
		fsSounds, err := c.model.ScanDirectory()
		if err != nil {
			log.FatalLog.Printf("Cron: error scanning sound directory: %v", err)
		}
		err = c.view.SyncDatabaseWithFileSystem(fsSounds)
		if err != nil {
			log.FatalLog.Printf("Cron: error syncing database with filesystem: %v", err)
		}
		//log.InfoLog.Println("Cron: database synced with filesystem")
	})
	// starts the scheduler asynchronously
	s.StartAsync()
	// starts the scheduler and blocks current execution path
	//s.StartBlocking()
}

// CheckBotActivity leaves the voice channel on every server where no sound
// was played for BOT_TIMEOUT minutes.
func (c *Controller) CheckBotActivity(s *discordgo.Session, m *model.Model) {
	botTimeout, err := strconv.Atoi(m.Config.BotTimeout)
	if err != nil {
		log.FatalLog.Fatalf("Failed to convert bot timeout to integer: %v", err)
	}
	timeout := time.Duration(botTimeout) * time.Minute

	scheduler := gocron.NewScheduler(time.UTC)
	_, _ = scheduler.Every(time.Minute).Do(func() {
		for _, g := range model.Guilds() {
			if !g.Playing() && time.Since(g.LastActivity()) > timeout {
				m.LeaveVoiceChannel(s, g.ID)
			}
		}
	})
	scheduler.StartAsync()
}

// CheckNewSounds pins the newest sounds in the BOT_CHANNEL of every server.
func (c *Controller) CheckNewSounds(s *discordgo.Session, m *model.Model) {
	scheduler := gocron.NewScheduler(time.UTC)
	_, _ = scheduler.Every(15 * time.Minute).Do(func() {
		for _, g := range model.Guilds() {
			m.PinNewSoundButtons(s, g.ID)
		}
	})
	scheduler.StartAsync()
}
