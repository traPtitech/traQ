package cmd

import (
	"github.com/spf13/cobra"
	"github.com/traPtitech/traQ/migration"
)

var dropDB bool

var migrateCommand = &cobra.Command{
	Use:   "migrate",
	Short: "Execute database schema migration only",
	RunE: func(cmd *cobra.Command, args []string) error {
		engine, err := getDatabase()
		if err != nil {
			return err
		}
		defer engine.Close()
		if dropDB {
			if err := migration.DropAll(engine); err != nil {
				return err
			}
		}
		return migration.Migrate(engine)
	},
}

func init() {
	migrateCommand.Flags().BoolVar(&dropDB, "reset", false, "whether to truncate database (drop all tables)")
}
