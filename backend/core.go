package backend

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/cyb3rplis/discord-bot-go/config"
	"github.com/cyb3rplis/discord-bot-go/controller"
	"github.com/cyb3rplis/discord-bot-go/db"
	log "github.com/cyb3rplis/discord-bot-go/logger"
	"github.com/cyb3rplis/discord-bot-go/model"
	"github.com/cyb3rplis/discord-bot-go/view"
)

func Init() {
	m, dbClose, err := db.InitModel()
	if err != nil {
		log.FatalLog.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := dbClose(); err != nil {
			log.ErrorLog.Printf("Failed to close database: %v", err)
		}
	}()
	modelInstance := model.New(&m)
	viewInstance := view.New(modelInstance)
	ctrl := controller.New(modelInstance, viewInstance)

	// Make sure the sound directory exists
	if _, err := os.Stat(modelInstance.Config.SoundsDir); os.IsNotExist(err) {
		if err = os.Mkdir(modelInstance.Config.SoundsDir, 0755); err != nil {
			log.FatalLog.Fatalf("Failed to create sound directory: %v", err)
		}
	}

	cfg := config.GetConfig()
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		log.FatalLog.Fatalf("error creating Discord session: %v", err)
	}

	dg.AddHandlerOnce(func(s *discordgo.Session, event *discordgo.Ready) {
		log.InfoLog.Printf("Bot is ready as %s, member of %d server(s)", s.State.User.Username, len(event.Guilds))
		view.BotReady = true
		// background jobs iterate over all known guilds, so they only start once
		ctrl.CheckBotActivity(s, modelInstance)
		ctrl.CheckNewSounds(s, modelInstance)
	})

	// GuildCreate fires once per server at startup, whenever the bot joins a new
	// server, and when a server becomes available again after a Discord outage.
	dg.AddHandler(guildCreate)
	dg.AddHandler(guildDelete)

	// All interaction handlers sit behind one guard: the bot only works inside
	// servers, so interactions without a guild/member (DMs) are ignored instead
	// of crashing on i.Member == nil.
	handlers := []func(*discordgo.Session, *discordgo.InteractionCreate){
		viewInstance.AutocompleteHandler,
		viewInstance.InteractionHandler,
		viewInstance.PromptInteractionButtons,
		viewInstance.PromptInteractionManage,
		viewInstance.PromptInteractionAudio,
		viewInstance.PromptInteractionFavorite,
		viewInstance.PromptInteractionGulag,
		viewInstance.PromptInteractionStats,
		viewInstance.PromptInteractionPlaySound,
		viewInstance.PromptInteractionMisc,
	}
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.GuildID == "" || i.Member == nil || i.Member.User == nil {
			return
		}
		model.Guild(i.GuildID) // make sure state exists
		for _, h := range handlers {
			go h(s, i)
		}
	})

	// Non-privileged intents only: GuildMembers/GuildPresences were only used
	// by the old user sync job and would require Discord verification once the
	// bot is on 100+ servers.
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMessages | discordgo.IntentsGuildVoiceStates

	// Open the websocket and begin listening.
	if err = dg.Open(); err != nil {
		log.ErrorLog.Fatalf("error opening Discord session: %v", err)
	}

	go ctrl.SyncSoundDirectories()

	// Wait here until CTRL-C or other term signal is received.
	log.InfoLog.Println("Bot is now running")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	_ = dg.Close()
	ctrl.Run(context.Background())
}

func NewBot() {
	Init()
}

// guildCreate registers a server and its slash commands.
func guildCreate(s *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild.Unavailable {
		log.WarningLog.Printf("Guild %s is unavailable, skipping", event.Guild.ID)
		return
	}
	model.RegisterGuild(event.Guild.ID, event.Guild.Name)
	log.InfoLog.Printf("Joined guild: %s (%s)", event.Guild.Name, event.Guild.ID)

	if err := view.RegisterCommands(s, event.Guild.ID); err != nil {
		log.ErrorLog.Println(err)
	}
}

// guildDelete forgets a server once the bot was removed from it.
func guildDelete(s *discordgo.Session, event *discordgo.GuildDelete) {
	if event.Unavailable {
		// outage, not a removal; the guild will come back via GuildCreate
		return
	}
	log.InfoLog.Printf("Removed from guild %s", event.ID)
	model.RemoveGuild(event.ID)
}
