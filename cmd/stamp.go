package cmd

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/spf13/cobra"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/utils/gormzap"
	"github.com/traPtitech/traQ/utils/twemoji"
	"go.uber.org/zap"
)

// stampCommand traQスタンプ操作コマンド
func stampCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "stamp",
		Short: "manage stamps",
	}

	cmd.AddCommand(
		stampInstallEmojisCommand(),
	)

	return &cmd
}

// stampInstallEmojisCommand ユニコード絵文字スタンプをインストールするコマンド
func stampInstallEmojisCommand() *cobra.Command {
	var update bool

	cmd := cobra.Command{
		Use:   "install-emojis",
		Short: "download and install Unicode emojiMeta stamps",
		Run: func(cmd *cobra.Command, args []string) {
			// Logger
			logger := getCLILogger()
			defer logger.Sync()

			// Database
			db, err := c.getDatabase()
			if err != nil {
				logger.Fatal("failed to connect database", zap.Error(err))
			}
			db.SetLogger(gormzap.New(logger.Named("gorm")))
			defer db.Close()

			// FileStorage
			fs, err := c.getFileStorage()
			if err != nil {
				logger.Fatal("failed to setup file storage", zap.Error(err))
			}

			// Repository
			repo, err := repository.NewGormRepository(db, hub.New(), logger)
			if err != nil {
				logger.Fatal("failed to initialize repository", zap.Error(err))
			}
			fm, err := file.InitFileManager(repo, fs, imaging.NewProcessor(provideImageProcessorConfig(c)), logger)
			if err != nil {
				logger.Fatal("failed to initialize file manager", zap.Error(err))
			}

			if err := twemoji.Install(repo, fm, logger, update); err != nil {
				logger.Fatal(err.Error())
			}
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&update, "update", false, "update(replace) existing Unicode emojiMeta stamp's image files")

	return &cmd
}
