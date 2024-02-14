package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/migration/v2tov3"
	"github.com/traPtitech/traQ/utils/gormzap"
)

// migrateV2ToV3Command traQv2データをv3データに変換するコマンド
// フロントエンドをRからSへと変更する場合、このコマンドでメッセージの埋め込み形式の変換が必要です
func migrateV2ToV3Command() *cobra.Command {
	var (
		dryRun             bool
		skipConvertMessage bool
		startMessagePage   int
		startFilePage      int
	)

	cmd := cobra.Command{
		Use:   "migrate-v2-to-v3",
		Short: "migrate from v2 to v3 (messages, files)",
		Run: func(_ *cobra.Command, _ []string) {
			// Logger
			logger := getCLILogger()
			defer logger.Sync()

			// Database
			db, err := c.getDatabase()
			if err != nil {
				logger.Fatal("failed to connect database", zap.Error(err))
			}
			db.Logger = gormzap.New(logger.Named("gorm"))
			sqlDB, err := db.DB()
			if err != nil {
				logger.Fatal("failed to get *sql.DB", zap.Error(err))
			}
			defer sqlDB.Close()

			if err := v2tov3.Run(db, logger, c.Origin, dryRun, startMessagePage, startFilePage, skipConvertMessage); err != nil {
				logger.Fatal(err.Error())
			}
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&dryRun, "dry-run", false, "dry run")
	flags.BoolVar(&skipConvertMessage, "skip-convert-message", false, "skip message converting")
	flags.IntVar(&startMessagePage, "start-message-page", 0, "start message page (zero-origin)")
	flags.IntVar(&startFilePage, "start-file-page", 0, "start file page (zero-origin)")

	return &cmd
}
