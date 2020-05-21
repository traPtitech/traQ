package cmd

import (
	"github.com/spf13/cobra"
	"github.com/traPtitech/traQ/migration"
)

// migrateCommand データベースマイグレーションコマンド
func migrateCommand() *cobra.Command {
	var dropDB bool

	cmd := cobra.Command{
		Use:   "migrate",
		Short: "Execute database schema migration only",
		RunE: func(cmd *cobra.Command, args []string) error {
			engine, err := c.getDatabase()
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

	flags := cmd.Flags()
	flags.BoolVar(&dropDB, "reset", false, "whether to truncate database (drop all tables)")

	return &cmd
}
