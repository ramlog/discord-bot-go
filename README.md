# discord-bot-go

<p align="center">
  <img src="./assets/discord-bot-logo2.webp" alt="discord-bot-go" width="350"/>
</p>

<p align="center">
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.23-00ADD8?logo=go&logoColor=white" alt="Go"></a>
  <a href="https://github.com/bwmarrin/discordgo"><img src="https://img.shields.io/badge/DiscordGo-v0.28.1-5865F2?logo=discord&logoColor=white" alt="DiscordGo"></a>
  <a href="https://hub.docker.com/r/cyb3rplis/discord-bot-go"><img src="https://img.shields.io/badge/Docker-Hub-2496ED?logo=docker&logoColor=white" alt="Docker Hub"></a>
  <a href="./LICENSE"><img src="https://img.shields.io/badge/License-MIT-green" alt="MIT License"></a>
</p>

A feature-rich [Discord](https://discord.com/) soundboard bot written in [Go](https://golang.org/) using the [DiscordGo](https://github.com/bwmarrin/discordgo/) library. Play and manage audio directly in Discord voice channels from local files or YouTube via [yt-dlp](https://github.com/yt-dlp/yt-dlp). Fully interactive with button-based controls — no commands needed.

## Screenshots

<!-- TODO: Add screenshots -->

| Feature | Preview |
|---------|---------|
| Category buttons | ![Category buttons](./assets/placeholder-categories.png) |
| Sound buttons | ![Sound buttons](./assets/placeholder-sounds.png) |
| Autocomplete | ![Autocomplete](./assets/placeholder-autocomplete.png) |
| Now playing | ![Now playing](./assets/placeholder-now-playing.png) |
| Now playing (YouTube) | ![Now playing YouTube](./assets/placeholder-now-playing-youtube.png) |
| Statistics | ![Statistics](./assets/placeholder-stats.png) |
| Favorites | ![Favorites](./assets/placeholder-favorites.png) |
| Favorite buttons | ![Favorite buttons](./assets/placeholder-favorite-buttons.png) |
| Gulag list | ![Gulag list](./assets/placeholder-gulag.png) |
| Sound management | ![Sound management](./assets/placeholder-manage.png) |

## Features

- **Soundboard** — Organize and play sounds from categorized local MP3 files
- **YouTube playback** — Stream audio from video platforms via yt-dlp
- **Button controls** — Interactive UI with no command-based interactions required
- **Categories** — Group sounds into categories for easy browsing
- **Favorites** — Add, remove, and list personal favorites
- **Statistics** — View sound and user stats, including a top 10 most-played leaderboard
- **Autocomplete** — Sound name suggestions as you type in commands
- **Sound management** — Add, delete, and move sounds; pin new sounds as messages
- **Gulag** — Temporarily restrict a user from using sounds for a specified duration
- **Spam protection** — Automatic rate limiting (max 15 interactions per 15 seconds)
- **Auto-leave** — Bot leaves voice channel after configurable inactivity timeout
- **Multi-server** — Add the bot to as many Discord servers as you like: all servers share the same sounds and favorites, statistics are kept per server

## Commands

### `/play`

Play a sound directly by name.

| Parameter | Required | Description |
|-----------|----------|-------------|
| `sound` | Yes | The name of the sound to play |

### `/random`

Play a random sound.

| Parameter | Required | Description |
|-----------|----------|-------------|
| `category` | No | Only pick from this category (autocomplete) |

### `/audio`

Play or replay audio from an external URL. Restricted to the bot moderators (`MOD_IDS`) and owners (`OWNER_IDS`), because it downloads arbitrary URLs on the host.

| Subcommand | Description |
|------------|-------------|
| `play` | Play audio from a URL (YouTube, Vimeo, etc.) |
| `last` | Replay the last played audio |

| Parameter | Required | Description |
|-----------|----------|-------------|
| `url` | Yes (for `play`) | The URL of the video to play |

### `/buttons`

Display the soundboard UI.

| Subcommand | Description |
|------------|-------------|
| `list` | List all sound category buttons in the channel |

### `/favorite`

Manage your personal favorite sounds.

| Subcommand | Description |
|------------|-------------|
| `buttons` | Display your favorite sounds as buttons |
| `add` | Add a sound to your favorites |
| `remove` | Remove a sound from your favorites |

| Parameter | Required | Description |
|-----------|----------|-------------|
| `sound` | Yes (for `add`/`remove`) | The name of the sound |

### `/manage`

Add, delete, or move sounds. Sounds are shared by all servers, so this command is restricted to the bot owners (`OWNER_IDS`).

| Subcommand | Description |
|------------|-------------|
| `create` | Create a new sound from a URL |
| `delete` | Delete a sound |
| `move` | Move a sound to a different category |

| Parameter | Required | Description |
|-----------|----------|-------------|
| `url` | Yes (for `create`) | URL to download audio from |
| `name` | Yes (for `create`/`delete`) | Name of the sound |
| `category` | Yes (for `create`/`move`) | Target category |
| `start_time` | No (for `create`) | Start timestamp for clipping |
| `end_time` | No (for `create`) | End timestamp for clipping |

### `/stats`

View usage statistics of the current server.

| Subcommand | Description |
|------------|-------------|
| `sounds` | Top 10 most played sounds |
| `users` | Top 10 most active users |
| `me` | Your personal sound statistics |

### `/gulag`

Restrict users from using the soundboard on all servers. Restricted to the bot owners (`OWNER_IDS`).

| Subcommand | Description |
|------------|-------------|
| `list` | List all users currently in the gulag |
| `add` | Add a user to the gulag |
| `remove` | Remove a user from the gulag |

| Parameter | Required | Description |
|-----------|----------|-------------|
| `user` | Yes (for `add`/`remove`) | The target user (Discord user picker) |
| `timeout` | Yes (for `add`) | Duration in minutes |

### `/misc`

Miscellaneous bot controls.

| Subcommand | Description |
|------------|-------------|
| `leave` | Force the bot to leave the voice channel |
| `version` | Show the running version, commit and build date |

## Button Interactions

The bot is designed around a button-driven interface. No commands are needed for everyday use.

| Button | Description |
|--------|-------------|
| **Category** | Click a category to browse its sounds |
| **Sound** | Click a sound to play it in your voice channel |
| **🎲 Random Sound** | Play a random sound (shown below the category buttons) |
| **Stop Sound** | Stop the current sound including all queued repeats |
| **🔁 Repeat** | Play the current sound once more after it finishes (max. 3 times per sound) |

<!-- TODO: Add screenshot of button interaction flow -->
![Button interaction flow](./assets/placeholder-button-flow.png)

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.23 |
| Discord library | [DiscordGo](https://github.com/bwmarrin/discordgo/) |
| Database | SQLite ([go-sqlite3](https://github.com/mattn/go-sqlite3)) |
| Scheduler | [gocron](https://github.com/go-co-op/gocron) |
| Audio encoding | [DCA](https://github.com/cyb3rplis/dca), FFmpeg |
| Media download | [yt-dlp](https://github.com/yt-dlp/yt-dlp) |
| Deployment | Docker, Docker Compose |
| CI/CD | GitHub Actions |

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)
- A [Discord bot token](https://discord.com/developers/applications)

## Quick Start

### 1. Configure environment variables

Copy the example file and fill in your values:

```bash
cp .env.example .env
```

| Variable | Description | Default |
|----------|-------------|---------|
| `TOKEN` | Discord bot token | *(required)* |
| `APP_PATH` | Path to the data directory | `./` |
| `OWNER_IDS` | Comma separated Discord user IDs that may use `/manage` and `/gulag` | *(required for admin commands)* |
| `MOD_IDS` | Comma separated Discord user IDs that may use `/audio` (owners always may) | *(empty = owners only)* |
| `BOT_TIMEOUT` | Minutes of inactivity before leaving voice | `120` |
| `BOT_CHANNEL` | Channel name where the bot pins the newest sounds (servers without it are skipped) | *(optional)* |
| `LEGACY_GUILD_ID` | Only for upgrading an existing install: server ID that the old statistics belong to | *(empty)* |

> Get user and server IDs by enabling *Developer Mode* in Discord (Settings → Advanced) and right-clicking a user or server → *Copy ID*.

### 2. Add sound files

Place MP3 files inside subdirectories of `${APP_PATH}/data/sounds/`. Each subdirectory acts as a **category**.

```
data/sounds/
├── memes/
│   ├── airhorn.mp3
│   └── bruh.mp3
└── music/
    └── intro.mp3
```

### 3. Run the bot

```bash
docker compose up -d
```

### 4. View logs

```bash
docker logs luren-bot
```

> **Note:** The `./data` directory also contains the `soundbot.db` SQLite database. Back up this directory regularly.

## Multiple servers

Invite the bot to any number of servers with the same invite link (Discord Developer Portal → OAuth2 → URL Generator, scopes `bot` and `applications.commands`; permissions: View Channels, Send Messages, Read Message History, Manage Messages, Pin Messages, Connect, Speak).

| Data | Scope |
|------|-------|
| Sounds & categories | Shared by all servers |
| Favorites | Per user, shared by all servers |
| Gulag | Per user, applies on all servers |
| Statistics (`/stats`) | Per server |
| Playback, stop button, auto-leave | Per server |

Slash commands are registered on every server automatically when the bot starts or joins a new server.

### Upgrading from the single-server version

The database is migrated automatically on first start. Existing statistics have no server assigned, so set `LEGACY_GUILD_ID` to the ID of the server the bot was running on before. If old statistics exist and the variable is missing, the bot refuses to start and leaves the database untouched. Back up `data/soundbot.db` before upgrading.

The variable `ADMIN_ROLE` is no longer used. The bot also no longer needs the privileged *Server Members* and *Presence* intents.

## Versioning

Every image built by GitHub Actions contains its version, visible via `/misc version` and in the container log on startup. The version comes from `git describe`:

- `v2.0.0` — built exactly from the tag `v2.0.0`
- `v2.0.0-3-gabc1234` — 3 commits after `v2.0.0`, at commit `abc1234`

To publish a release: `git tag v2.1.0 && git push origin v2.1.0`. This also creates the image tag `ghcr.io/<owner>/discord-bot-go:2.1.0`, which you can pin in `compose.yml`.

## Updating

Pull the latest image and restart:

```bash
docker compose pull
docker compose up -d
```

## Stopping

```bash
docker compose stop
```

## Restart Policy

The default `compose.yml` uses `restart: unless-stopped`, meaning the container will automatically restart on reboot unless manually stopped. See the [Docker restart policy docs](https://docs.docker.com/config/containers/start-containers-automatically/) for other options.

## Development

Build and run with live reload on code changes:

```bash
docker compose -f compose.dev.yml up -d --watch --build
```

The container rebuilds automatically when changes are detected in the backend.

## License

This project is licensed under the [MIT License](./LICENSE).
