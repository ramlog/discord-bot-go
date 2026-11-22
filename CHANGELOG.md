# Changelog

All notable changes since forking [cyb3rplis/discord-bot-go](https://github.com/cyb3rplis/discord-bot-go) at commit `843136b`.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), versions follow [Semantic Versioning](https://semver.org/).

## [2.3.0] - 2026-11-22

### Added
- `/help` lists all commands and buttons with short explanations. Only the requesting user sees it, and moderator/owner commands are only listed for mods and owners.

## [2.2.0] - 2026-11-22

### Added
- `/random` plays a random sound, optionally limited to one `category` (with autocomplete).
- **🎲 Random Sound** button below the category buttons of `/buttons list`.
- **🔁 Repeat** button next to *Stop Sound*: queues the current sound once more, up to 3 times per sound. The button shows the count (`1/3` … `3/3`) and is disabled at the limit. Every repeat counts in the statistics of the user who clicked.

### Changed
- *Stop Sound* now cancels the current sound **and all queued repeats**. Starting another sound also discards the repeats of the previous one.
- `/play`, sound buttons, `/random`, the random button and `/audio` share one playback function instead of three copies of the same code.

## [2.1.1] - 2026-11-22

### Security
- Central spam protection for **all** slash commands and buttons (previously buttons only): more than 15 interactions in 15 seconds put the user in the gulag for one minute. The warning is only visible to the user.
- Panic recovery for all interaction handlers and cron jobs. A runtime error no longer takes the bot offline on every server.
- `/play` checks that the sound exists before announcing it in the channel. Free text like `@everyone` or masked links can no longer be posted through the bot.
- Bot messages may only mention individual users, never `@everyone`, `@here` or roles.
- `yt-dlp` runs with `--no-playlist`, `--max-filesize 100M` and `--ignore-config`.

### Fixed
- Database errors in category and sound lookups no longer terminate the process.
- SQLite waits up to 5 seconds for a lock (`_busy_timeout`) instead of failing with `database is locked` when several servers write at once.
- The bot token is no longer partially printed to the log (this also crashed on tokens shorter than 10 characters).

## [2.1.0] - 2026-11-22

### Added
- `MOD_IDS` environment variable: comma separated Discord user IDs allowed to use `/audio`. Owners are always allowed. `/audio` downloads arbitrary URLs on the host and is therefore no longer available to everyone.

## [2.0.0] - 2026-11-22

### Added
- **Multi-server support:** the bot can be invited to any number of Discord servers.
  - Shared by all servers: sounds, categories, favorites, gulag.
  - Per server: statistics (`/stats`), playback, stop button, voice inactivity timeout, temporary `/audio` file.
- Slash commands are registered automatically on every server at startup and when the bot joins a new server (bulk overwrite, which also removes outdated commands).
- `OWNER_IDS` environment variable: only these users may use `/manage` and `/gulag`, because both affect all servers.
- Versioned database migrations (`PRAGMA user_version`). Migration 1 adds `guild_id` to `stats_users`; existing statistics are assigned to the server in `LEGACY_GUILD_ID`. Without it the bot refuses to start and leaves the database untouched.
- `/misc version` shows version, commit (linked), build date, Go version, uptime and number of servers. The version is derived from `git describe` and injected at build time.
- GitHub Actions builds and pushes the image to the GitHub Container Registry (`ghcr.io/<owner>/discord-bot-go`) with the tags `latest`, the release version (e.g. `2.2.0`) and `sha-<commit>`. No secrets required.

### Changed
- `/gulag` identifies users by Discord user ID instead of display name and uses Discord's user picker. The timeout is an integer option.
- `/manage` and `/gulag` are hidden from regular members by default.
- `/play` now counts in the statistics (previously only button clicks did).
- Users without a global display name are stored with their username.
- The bot no longer needs the privileged *Server Members* and *Presence* intents (the user sync job was removed).
- `.env.example` uses comments on separate lines. Inline comments after empty values broke `docker compose`.

### Removed
- `ADMIN_ROLE` (was never checked).

### Security
- **Command injection:** `yt-dlp`, `ffmpeg` and `dca` were started via `bash -c` with user input inside the command string. They are now started directly with argument lists, so input like `; rm -rf /` is never executed. Only `http(s)` URLs are accepted.
- Sound and category names from `/manage` are validated (letters, digits, space, `-`, `_`, max. 40 characters) to prevent path traversal.

### Fixed
- The stop button deleted the wrong message (used the interaction ID instead of the message ID).
- The inactivity timer could disconnect the bot in the middle of a long `/audio`.
- Sound files with spaces in their name failed to convert.
- `/manage create` gave no feedback when download or conversion failed.
- Interactions from DMs no longer crash the bot.

[2.3.0]: https://github.com/ramlog/discord-bot-go/compare/v2.2.0...v2.3.0
[2.2.0]: https://github.com/ramlog/discord-bot-go/compare/v2.1.1...v2.2.0
[2.1.1]: https://github.com/ramlog/discord-bot-go/compare/v2.1.0...v2.1.1
[2.1.0]: https://github.com/ramlog/discord-bot-go/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/ramlog/discord-bot-go/compare/843136b...v2.0.0
