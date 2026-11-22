package db

import (
	"database/sql"
	"fmt"

	log "github.com/cyb3rplis/discord-bot-go/logger"
)

// migration is one schema change. Migrations run in order, exactly once,
// tracked via SQLite's built-in `PRAGMA user_version` counter.
type migration struct {
	description string
	apply       func(tx *sql.Tx) error
}

// migrations must only ever be appended to, never reordered or edited,
// because existing databases remember how many of them already ran.
var migrations = []migration{
	{
		description: "multi-server: add guild_id to stats_users",
		apply:       migrateStatsPerGuild,
	},
}

// runMigrations brings the database schema up to the latest version.
func runMigrations(db *sql.DB) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	for i := version; i < len(migrations); i++ {
		m := migrations[i]
		log.InfoLog.Printf("Running database migration %d: %s", i+1, m.description)

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := m.apply(tx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %d (%s): %w", i+1, m.description, err)
		}
		// PRAGMA does not support bind parameters, the value is our own int.
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// migrateStatsPerGuild rebuilds stats_users with a guild_id column.
// SQLite cannot change a UNIQUE constraint in place, so the table is
// recreated and the existing rows are copied over.
func migrateStatsPerGuild(tx *sql.Tx) error {
	var existingRows int
	if err := tx.QueryRow("SELECT COUNT(*) FROM stats_users").Scan(&existingRows); err != nil {
		return err
	}

	legacyGuild := Config.LegacyGuildID
	if existingRows > 0 && legacyGuild == "" {
		return fmt.Errorf("found %d existing statistic rows but LEGACY_GUILD_ID is not set; "+
			"set it to the ID of the server the bot was running on so the old stats can be assigned to it", existingRows)
	}

	stmts := []string{
		`CREATE TABLE stats_users_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id TEXT NOT NULL,
			user_id INTEGER NOT NULL,
			sound_id INTEGER NOT NULL,
			count INTEGER DEFAULT 0,
			UNIQUE(guild_id, user_id, sound_id),
			CONSTRAINT fk_sounds FOREIGN KEY (sound_id) REFERENCES sounds(id) ON DELETE CASCADE,
			CONSTRAINT fk_users FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}
	for _, q := range stmts {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}

	if existingRows > 0 {
		res, err := tx.Exec(`INSERT INTO stats_users_new (guild_id, user_id, sound_id, count)
			SELECT ?, user_id, sound_id, count FROM stats_users`, legacyGuild)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		log.InfoLog.Printf("Migrated %d statistic rows to guild %s", n, legacyGuild)
	}

	for _, q := range []string{
		`DROP TABLE stats_users`,
		`ALTER TABLE stats_users_new RENAME TO stats_users`,
		`CREATE INDEX IF NOT EXISTS idx_stats_users_guild ON stats_users(guild_id)`,
	} {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
