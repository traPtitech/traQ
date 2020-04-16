package cmd

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/spf13/cobra"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/gormzap"
	"go.uber.org/zap"
)

// fileCommand traQ管理ファイル操作コマンド
func fileCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "file",
		Short: "manage files",
	}

	cmd.AddCommand(
		filePruneCommand(),
	)

	flags := cmd.PersistentFlags()
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

	return &cmd
}

// filePruneCommand 未使用ファイル解放コマンド
func filePruneCommand() *cobra.Command {
	var (
		dryRun   bool
		userFile bool
	)

	cmd := cobra.Command{
		Use:   "prune",
		Short: "delete files which are not used or linked to anywhere",
		Run: func(cmd *cobra.Command, args []string) {
			// Logger
			logger := getLogger()
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
			repo, err := repository.NewGormRepository(db, fs, hub.New(), logger.Named("repository"))
			if err != nil {
				logger.Fatal("failed to initialize repository", zap.Error(err))
			}

			// 未使用アイコン・スタンプ画像ファイル列挙
			var (
				files []*model.File
				tmp   []*model.File
			)
			if err := db.
				Where("id NOT IN ?", db.Table("users").Select("icon").SubQuery()).
				Where(model.File{Type: model.FileTypeIcon}).
				Find(&tmp).
				Error; err != nil {
				logger.Fatal(err.Error())
			}
			files = append(files, tmp...)
			tmp = nil
			if err := db.
				Where("id NOT IN ?", db.Table("stamps").Select("file_id").SubQuery()).
				Where(model.File{Type: model.FileTypeStamp}).
				Find(&tmp).
				Error; err != nil {
				logger.Fatal(err.Error())
			}
			files = append(files, tmp...)
			tmp = nil

			// 未使用ユーザーアップロードファイル
			if userFile {
				// TODO
			}

			logger.Sugar().Infof("%d unused-files was detected", len(files))
			for _, file := range files {
				logger.Sugar().Infof("%s - %s", file.ID, file.CreatedAt)
				if !dryRun {
					if err := repo.DeleteFile(file.ID); err != nil {
						logger.Fatal(err.Error())
					}
				}
			}
		},
	}

	flags := cmd.Flags()
	flags.BoolVar(&dryRun, "dry-run", false, "list target files only (no delete)")
	flags.BoolVar(&userFile, "include-user-file", false, "include user-uploaded files which has no link to any messages (may take long time)")

	return &cmd
}
