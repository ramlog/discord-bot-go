package view

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// adminOnly hides a command from regular members by default. Server admins can
// change that under Server Settings > Integrations. The real security check is
// the OWNER_IDS check at execution time (see requireOwner).
var adminOnly = func() *int64 { p := int64(discordgo.PermissionAdministrator); return &p }()

// guildOnly makes commands unavailable in DMs.
var guildOnly = &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild}

// AllCommands returns every slash command of the bot.
func AllCommands() []*discordgo.ApplicationCommand {
	cmds := []*discordgo.ApplicationCommand{
		commandAudio(),
		commandButtons(),
		commandManage(),
		commandFavorite(),
		commandGulag(),
		commandStats(),
		commandPlaySound(),
		commandRandom(),
		commandMisc(),
	}
	for _, c := range cmds {
		c.Contexts = guildOnly
	}
	return cmds
}

// RegisterCommands writes all slash commands to one guild. BulkOverwrite is
// idempotent: it creates new commands, updates changed ones and deletes
// commands that no longer exist, so it is safe to call on every GuildCreate.
func RegisterCommands(s *discordgo.Session, guildID string) error {
	_, err := s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildID, AllCommands())
	if err != nil {
		return fmt.Errorf("registering commands in guild %s: %w", guildID, err)
	}
	return nil
}

// RegisterPromptInteractionsAudio - Register prompt interactions
func commandAudio() *discordgo.ApplicationCommand {
	commandName := "audio"
	// Register the command with subcommands for URL and last
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        commandName,
			Description: "Play audio from a URL or replay the last audio (mods only)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "play",
					Description: "Play audio from a URL (e.g., YouTube, Vimeo, etc.)",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "url",
							Description: "The URL of the video you want to play",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "last",
					Description: "Replay the last played audio",
				},
			},
		},
	}
	return commands[0]
}

// RegisterPromptInteractionsButtons - Register prompt interactions
func commandButtons() *discordgo.ApplicationCommand {
	commandName := "buttons"
	// Register the command globally
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        commandName,
			Description: "Show the soundboard",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List all sound category buttons",
				},
			},
		},
	}
	return commands[0]
}

// RegisterPromptInteractionsDelete - Register prompt interactions
func commandManage() *discordgo.ApplicationCommand {
	commandName := "manage"
	// Register the command globally
	commands := []*discordgo.ApplicationCommand{
		{
			Name:                     commandName,
			Description:              "Manage sounds on ALL servers (bot owners only)",
			DefaultMemberPermissions: adminOnly,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "create",
					Description: "Create a new sound button using a URL (e.G. YouTube, Vimeo, etc.)",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "url",
							Description: "The URL of the video you want to create a sound button for",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "The name of the sound button",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "category",
							Description: "The category of the sound button",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "start_time",
							Description: "The start time of the sound button",
							Required:    false,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "end_time",
							Description: "The end time of the sound button",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "delete",
					Description: "Delete a sound button by name",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "name",
							Description:  "The name of the sound button to delete",
							Required:     true,
							Autocomplete: true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "move",
					Description: "Move a sound to a different category",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "name",
							Description:  "The name of the sound button to move",
							Required:     true,
							Autocomplete: true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "category",
							Description: "The new category the sound button should be moved to",
							Required:    true,
						},
					},
				},
			},
		},
	}
	return commands[0]
}

// RegisterPromptInteractionsFavorite - Register prompt interactions for favorite
func commandFavorite() *discordgo.ApplicationCommand {
	commandName := "favorite"
	// Register the command globally
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        commandName,
			Description: "Manage your favorite sounds",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "buttons",
					Description: "List all your favorite sound buttons",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add a sound to your favorites",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "sound",
							Description:  "The name of the sound to add",
							Required:     true,
							Autocomplete: true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove a sound from your favorites",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:         discordgo.ApplicationCommandOptionString,
							Name:         "sound",
							Description:  "The name of the sound to remove",
							Required:     true,
							Autocomplete: true,
						},
					},
				},
			},
		},
	}
	return commands[0]
}

// RegisterPromptInteractionsGulag - Register prompt interactions for gulag
func commandGulag() *discordgo.ApplicationCommand {
	commandName := "gulag"
	// Register the command globally
	commands := []*discordgo.ApplicationCommand{
		{
			Name:                     commandName,
			Description:              "Block users from sounds on ALL servers (bot owners only)",
			DefaultMemberPermissions: adminOnly,
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List all users in the gulag",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add a user to the gulag",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to add to the gulag",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "timeout",
							Description: "Time in minutes to keep the user in the gulag",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove a user from the gulag",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "The user to remove from the gulag",
							Required:    true,
						},
					},
				},
			},
		},
	}
	return commands[0]
}

// RegisterPromptInteractionsStats - Register prompt interactions for stats
func commandStats() *discordgo.ApplicationCommand {
	commandName := "stats"
	// Register the command globally
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        commandName,
			Description: "Get statistics for sounds and users",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "sounds",
					Description: "Get statistics for top sounds",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "users",
					Description: "Get statistics for top users",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "me",
					Description: "Get your personal sound statistics",
				},
			},
		},
	}
	return commands[0]
}

func commandPlaySound() *discordgo.ApplicationCommand {
	commandName := "play"
	// Register the command globally
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        commandName,
			Description: "Play a sound by name",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "sound",
					Description:  "The name of the sound to play",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
	}
	return commands[0]
}

func commandMisc() *discordgo.ApplicationCommand {
	commandName := "misc"
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        commandName,
			Description: "Miscellaneous commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "leave",
					Description: "Leave the voice channel",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "version",
					Description: "Show which version of the bot is running",
				},
			},
		},
	}
	return commands[0]
}

func commandRandom() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:        "random",
		Description: "Play a random sound",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:         discordgo.ApplicationCommandOptionString,
				Name:         "category",
				Description:  "Only pick from this category",
				Required:     false,
				Autocomplete: true,
			},
		},
	}
}
