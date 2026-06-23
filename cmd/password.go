package cmd

import (
	"fmt"

	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/spf13/cobra"
)

var passwordCmd = &cobra.Command{
	Use:   "password",
	Short: "Password administration",
}

func init() {
	rootCmd.AddCommand(passwordCmd)
}

// fatalDatabase logs a database error and exits, hinting at file permissions
// when the database is not writable by the current user.
func fatalDatabase(err error) {
	if db.IsReadonly(err) {
		log.FATAL.Println("database is not writable; run this command as the user owning the database file, e.g. sudo -u evcc")
	}
	log.FATAL.Fatal(err)
}

// persistPasswordSettings writes pending settings to the database
func persistPasswordSettings() {
	if err := settings.Persist(); err != nil {
		fatalDatabase(fmt.Errorf("cannot save settings: %w", err))
	}
}
