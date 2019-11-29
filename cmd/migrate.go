package cmd

import (
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/spf13/cobra"
	"github.com/traPtitech/traQ/migration"
)

var migrateCommand = &cobra.Command{
	Use:   "migrate",
	Short: "Execute database schema migration only",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := getDatabase()
		if err != nil {
			return err
		}
		defer engine.Close()
		return migration.Migrate(engine)
	},
}
