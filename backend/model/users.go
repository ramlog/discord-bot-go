package model

import (
	"github.com/bwmarrin/discordgo"
	"github.com/cyb3rplis/discord-bot-go/config"
)

// DisplayName returns the best human readable name of a Discord user.
func DisplayName(user *discordgo.User) string {
	if user.GlobalName != "" {
		return user.GlobalName
	}
	return user.Username
}

// GetUsers returns all users from the database
func (m *Model) GetUsers() (users []config.ExtendedUser, err error) {
	rows, err := m.Db.Query("SELECT id, username, gulagged FROM users;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var u config.ExtendedUser
		u.User = &discordgo.User{}
		if err = rows.Scan(&u.User.ID, &u.User.GlobalName, &u.Gulagged); err != nil {
			return nil, err
		}
		users = append(users, u)
	}

	return users, rows.Err()
}

// AddUser adds a user to the database (users are global across all servers)
func (m *Model) AddUser(user *discordgo.User) error {
	_, err := m.Db.Exec("INSERT INTO users (id, username) VALUES (?, ?) ON CONFLICT(id) DO UPDATE SET username = excluded.username;", user.ID, DisplayName(user))
	return err
}

// SetUserGulaggedValue loads the gulag state of a user (by Discord user ID).
// Returns sql.ErrNoRows if the user is not known yet.
func (m *Model) SetUserGulaggedValue(user *discordgo.User) (config.ExtendedUser, error) {
	extendedUser := config.ExtendedUser{
		User: user,
	}
	err := m.Db.QueryRow("SELECT gulagged FROM users WHERE id = ?;", user.ID).Scan(&extendedUser.Gulagged)
	return extendedUser, err
}
