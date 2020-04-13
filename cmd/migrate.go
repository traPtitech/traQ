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
	flags.String("host", "", "database host")
	bindPFlag(flags, "mariadb.host", "host")
	flags.Int("port", 0, "database port")
	bindPFlag(flags, "mariadb.port", "port")
	flags.String("name", "", "database name")
	bindPFlag(flags, "mariadb.database", "name")
	flags.String("user", "", "database user")
	bindPFlag(flags, "mariadb.username", "user")
	flags.String("pass", "", "database password")
	bindPFlag(flags, "mariadb.password", "pass")
	flags.BoolVar(&dropDB, "reset", false, "whether to truncate database (drop all tables)")

	return &cmd
}
