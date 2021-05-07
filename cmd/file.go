package cmd

import (
	"fmt"
	"github.com/leandro-lugaresi/hub"
	"github.com/spf13/cobra"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/utils/gormzap"
	"go.uber.org/zap"
	"io"
)

// fileCommand traQ管理ファイル操作コマンド
func fileCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "file",
		Short: "manage files",
	}

	cmd.AddCommand(
		filePruneCommand(),
		genWaveform(),
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

			// Repository
			repo, err := repository.NewGormRepository(db, hub.New(), logger)
			if err != nil {
				logger.Fatal("failed to initialize repository", zap.Error(err))
			}

			// FileManager
			fm, err := file.InitFileManager(repo, fs, imaging.NewProcessor(provideImageProcessorConfig(c)), logger)
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

// genWaveform 波形画像生成コマンド
func genWaveform() *cobra.Command {
	canGenerateWaveform := func(mimeType string) bool {
		switch mimeType {
		case "audio/mpeg", "audio/mp3", "audio/wav", "audio/x-wav":
			return true
		default:
			return false
		}
	}

	return &cobra.Command{
		Use:   "gen-waveform",
		Short: "Generate waveform thumbnail for old uploads",
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

			// ImageProcessor
			ip := imaging.NewProcessor(provideImageProcessorConfig(c))

			// FileManager
			fm, err := file.InitFileManager(repo, fs, ip, logger)
			if err != nil {
				logger.Fatal("failed to initialize file manager", zap.Error(err))
			}

			generateWaveform := func(file model.File) error {
				fid := file.GetID()

				src, err := file.Open()
				if err != nil {
					return fmt.Errorf("failed to open file: %w", err)
				}
				defer src.Close()

				const (
					waveformWidth  = 1280
					waveformHeight = 540
				)

				var r io.Reader
				switch file.GetMIMEType() {
				case "audio/mpeg", "audio/mp3":
					r, err = ip.WaveformMp3(src, waveformWidth, waveformHeight)
					if err != nil {
						return fmt.Errorf("failed to generate thumbnail: %w", err)
					}
				case "audio/wav", "audio/x-wav":
					r, err = ip.WaveformWav(src, waveformWidth, waveformHeight)
					if err != nil {
						return fmt.Errorf("failed to generate thumbnail: %w", err)
					}
				default:
					return nil
				}

				thumbnail := model.FileThumbnail{
					FileID: fid,
					Type:   model.ThumbnailTypeWaveform,
					Mime:   "image/svg+xml",
					Width:  waveformWidth,
					Height: waveformHeight,
				}
				if err := db.Create(thumbnail).Error; err != nil {
					return fmt.Errorf("failed to save file thumbnail to db: %w", err)
				}

				key := fid.String() + "-" + model.ThumbnailTypeWaveform.Suffix()
				if err := fs.SaveByKey(r, key, key+".svg", "image/svg+xml", model.FileTypeThumbnail); err != nil {
					if err := db.Delete(thumbnail).Error; err != nil {
						logger.Error("failed to rollback file thumbnail info on db", zap.Error(err), zap.Stringer("fid", fid))
					}
					return fmt.Errorf("failed to save thumbnail to storage: %w", err)
				}

				return nil
			}

			const batch = 100
			// counter variables
			var (
				offset  = 0
				total   = 0
				success = 0
			)
			// run
			for {
				files, more, err := fm.List(repository.FilesQuery{
					Limit:  batch,
					Offset: offset,
					Asc:    false,
					Type:   model.FileTypeUserFile,
				})
				if err != nil {
					logger.Fatal("failed to list files", zap.Error(err))
				}

				logger.Info(fmt.Sprintf("listing files from %d to %d", offset, offset+len(files)-1))

				for _, f := range files {
					if !canGenerateWaveform(f.GetMIMEType()) {
						continue
					}
					if ok, _ := f.GetThumbnail(model.ThumbnailTypeWaveform); ok {
						// already has waveform thumbnail
						continue
					}

					// generate waveform
					total++
					if err := generateWaveform(f); err != nil {
						logger.Error("failed to generate waveform", zap.Error(err), zap.Stringer("fid", f.GetID()))
						continue
					}
					success++
				}

				if !more {
					break
				}
				offset += batch

				logger.Info(fmt.Sprintf("generating waveform images: %d succeeded out of %d total attempts", success, total))
			}

			logger.Info(fmt.Sprintf("finished generating waveform images: %d succeeded out of %d total attempts", success, total))
		},
	}
}
