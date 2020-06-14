package cmd

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/spf13/cobra"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/file"
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

			// Repository チャンネルツリーを作らないので注意
			repo, err := repository.NewGormRepository(db, hub.New(), logger)
			if err != nil {
				logger.Fatal("failed to initialize repository", zap.Error(err))
			}

			fm, err := file.InitFileManager(repo, fs, logger)
			if err != nil {
				logger.Fatal("failed to initialize file manager", zap.Error(err))
			}

			// 未使用アイコン・スタンプ画像ファイル列挙
			var (
				files []*model.FileMeta
				tmp   []*model.FileMeta
			)
			if err := db.
				Where("id NOT IN ?", db.Table("users").Select("icon").SubQuery()).
				Where(model.FileMeta{Type: model.FileTypeIcon}).
				Find(&tmp).
				Error; err != nil {
				logger.Fatal(err.Error())
			}
			files = append(files, tmp...)
			tmp = nil
			if err := db.
				Where("id NOT IN ?", db.Table("stamps").Select("file_id").SubQuery()).
				Where(model.FileMeta{Type: model.FileTypeStamp}).
				Find(&tmp).
				Error; err != nil {
				logger.Fatal(err.Error())
			}
			files = append(files, tmp...)
			tmp = nil

			// 未使用ユーザーアップロードファイル
			if userFile {
				// TODO 実装
				logger.Warn("include-user-file flag is not implemented currently")
			}

			logger.Sugar().Infof("%d unused-files was detected", len(files))
			for _, file := range files {
				logger.Sugar().Infof("%s - %s", file.ID, file.CreatedAt)
				if !dryRun {
					if err := fm.Delete(file.ID); err != nil {
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
