package model

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

// GulagUser blocks a user from playing sounds on all servers for the given minutes.
func (m *Model) GulagUser(user *discordgo.User, minutes int) error {
	// make sure the user exists, otherwise the UPDATE below has nothing to hit
	if err := m.AddUser(user); err != nil {
		return err
	}

	timeout := "+" + strconv.Itoa(minutes) + " minutes"
	res, err := m.Db.Exec("UPDATE users SET gulagged = DATETIME(CURRENT_TIMESTAMP, ?) WHERE id = ?;", timeout, user.ID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// ReleaseUser removes the gulag timeout from a user
func (m *Model) ReleaseUser(user *discordgo.User) error {
	_, err := m.Db.Exec("UPDATE users SET gulagged = NULL WHERE id = ?;", user.ID)
	return err
}
