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
		RunE: func(_ *cobra.Command, _ []string) error {
			engine, err := c.getDatabase()
			if err != nil {
				return err
			}
			db, err := engine.DB()
			if err != nil {
				return err
			}
			defer db.Close()
			if dropDB {
				if err := migration.DropAll(engine); err != nil {
					return err
				}
			}
			_, err = migration.Migrate(engine)
			return err
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&dropDB, "reset", false, "whether to truncate database (drop all tables)")

	return &cmd
}
