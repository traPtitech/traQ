package cmd

import (
	"fmt"
	"image/png"
	"io"
	"time"

	"github.com/leandro-lugaresi/hub"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/repository/gorm"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/utils/gormzap"
	"github.com/traPtitech/traQ/utils/optional"
)

// fileCommand traQ管理ファイル操作コマンド
func fileCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "file",
		Short: "manage files",
	}

	cmd.AddCommand(
		filePruneCommand(),
		genMissingThumbnails(),
		genGroupImages(),
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
			db.Logger = gormzap.New(logger.Named("gorm"))
			sqlDB, err := db.DB()
			if err != nil {
				logger.Fatal("failed to get *sql.DB", zap.Error(err))
			}
			defer sqlDB.Close()

			// FileStorage
			fs, err := c.getFileStorage()
			if err != nil {
				logger.Fatal("failed to setup file storage", zap.Error(err))
			}

			// Repository
			repo, _, err := gorm.NewGormRepository(db, hub.New(), logger, false)
			if err != nil {
				logger.Fatal("failed to initialize repository", zap.Error(err))
			}

			// FileManager
			fm, err := file.InitFileManager(repo, fs, imaging.NewProcessor(provideImageProcessorConfig(&c)), logger)
			if err != nil {
				logger.Fatal("failed to initialize file manager", zap.Error(err))
			}

			// 未使用アイコン・スタンプ画像ファイル列挙
			var (
				files []*model.FileMeta
				tmp   []*model.FileMeta
			)
			if err := db.
				Where("id NOT IN (?)", db.Table("users").Select("icon")).
				Where("id NOT IN (?)", db.Table("user_groups").Select("icon")).
				Where(model.FileMeta{Type: model.FileTypeIcon}).
				Find(&tmp).
				Error; err != nil {
				logger.Fatal(err.Error())
			}
			files = append(files, tmp...)
			tmp = nil
			if err := db.
				Where("id NOT IN (?)", db.Table("stamps").Select("file_id")).
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

// genMissingThumbnails 不足サムネイル生成コマンド
func genMissingThumbnails() *cobra.Command {
	canGenerateImageThumb := func(mimeType string) bool {
		switch mimeType {
		case "image/jpeg", "image/png", "image/gif", "image/webp":
			return true
		default:
			return false
		}
	}
	canGenerateWaveform := func(mimeType string) bool {
		switch mimeType {
		case "audio/mpeg", "audio/mp3", "audio/wav", "audio/x-wav":
			return true
		default:
			return false
		}
	}

	return &cobra.Command{
		Use:   "gen-missing-thumbs",
		Short: "Generate missing thumbnails",
		Run: func(cmd *cobra.Command, args []string) {
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

			// FileStorage
			fs, err := c.getFileStorage()
			if err != nil {
				logger.Fatal("failed to setup file storage", zap.Error(err))
			}

			// ImageProcessor
			ip := imaging.NewProcessor(provideImageProcessorConfig(&c))

			generateImageThumb := func(file *model.FileMeta) error {
				fid := file.ID

				src, err := fs.OpenFileByKey(file.ID.String(), file.Type)
				if err != nil {
					return fmt.Errorf("failed to open file: %w", err)
				}
				defer src.Close()

				thumb, err := ip.Thumbnail(src)
				if err != nil {
					return fmt.Errorf("failed to generate thumbnail: %w", err)
				}

				thumbnail := model.FileThumbnail{
					FileID: fid,
					Type:   model.ThumbnailTypeImage,
					Mime:   "image/png",
					Width:  thumb.Bounds().Size().X,
					Height: thumb.Bounds().Size().Y,
				}
				if err := db.Create(thumbnail).Error; err != nil {
					return fmt.Errorf("failed to save file thumbnail to db: %w", err)
				}

				r, w := io.Pipe()
				go func() {
					defer w.Close()
					_ = png.Encode(w, thumb)
				}()

				key := file.ID.String() + "-" + model.ThumbnailTypeImage.Suffix()
				if err := fs.SaveByKey(r, key, key+".png", "image/png", model.FileTypeThumbnail); err != nil {
					if err := db.Delete(thumbnail).Error; err != nil {
						logger.Error("failed to rollback file thumbnail info on db", zap.Error(err), zap.Stringer("fid", fid))
					}
					return fmt.Errorf("failed to save thumbnail to storage: %w", err)
				}

				return nil
			}
			generateWaveform := func(file *model.FileMeta) error {
				fid := file.ID

				src, err := fs.OpenFileByKey(file.ID.String(), file.Type)
				if err != nil {
					return fmt.Errorf("failed to open file: %w", err)
				}
				defer src.Close()

				const (
					waveformWidth  = 1280
					waveformHeight = 540
				)

				var r io.Reader
				switch file.Mime {
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
				lastCreatedAt     = time.Time{}
				total             = 0
				imageThumbTotal   = 0
				imageThumbSuccess = 0
				waveformTotal     = 0
				waveformSuccess   = 0
			)
			// run
			for {
				var files []*model.FileMeta
				err = db.Raw("SELECT f.* FROM files f "+
					"LEFT JOIN files_thumbnails ft on f.id = ft.file_id "+
					"WHERE f.type = '' AND f.deleted_at IS NULL AND f.created_at > ? "+
					"AND f.mime IN ("+
					// サムネイル生成が可能なmimeが変わったらここを変える
					"'image/jpeg', 'image/png', 'image/gif', 'image/webp', "+
					"'audio/mpeg', 'audio/mp3', 'audio/wav', 'audio/x-wav'"+
					") "+
					"GROUP BY f.id, f.created_at "+
					"HAVING COUNT(ft.file_id) = 0 "+
					"ORDER BY f.created_at "+
					"LIMIT ?", lastCreatedAt, batch).
					Scan(&files).Error
				if err != nil {
					logger.Fatal("failed to list files", zap.Error(err))
				}

				logger.Info(fmt.Sprintf("listing files from %d to %d", total, total+len(files)-1))

				for _, f := range files {
					lastCreatedAt = f.CreatedAt

					// generate image thumbnail
					if canGenerateImageThumb(f.Mime) {
						imageThumbTotal++
						if err := generateImageThumb(f); err != nil {
							logger.Error("failed to generate image thumbnail", zap.Error(err), zap.Stringer("fid", f.ID))
						} else {
							imageThumbSuccess++
						}
					}
					// generate waveform
					if canGenerateWaveform(f.Mime) {
						waveformTotal++
						if err := generateWaveform(f); err != nil {
							logger.Error("failed to generate waveform", zap.Error(err), zap.Stringer("fid", f.ID))
						} else {
							waveformSuccess++
						}
					}
				}

				if len(files) < batch {
					break
				}
				total += batch

				logger.Info(fmt.Sprintf("generating missing thumbnails: images success / total (%d / %d), waveform success / total (%d / %d)", imageThumbSuccess, imageThumbTotal, waveformSuccess, waveformTotal))
			}

			logger.Info(fmt.Sprintf("finished generating missing thumbnails: images success / total (%d / %d), waveform success / total (%d / %d)", imageThumbSuccess, imageThumbTotal, waveformSuccess, waveformTotal))
		},
	}
}

// genGroupImages ユーザーグループアイコン生成コマンド
func genGroupImages() *cobra.Command {
	return &cobra.Command{
		Use:   "gen-group-images",
		Short: "Generate missing icons for user groups",
		Run: func(cmd *cobra.Command, args []string) {
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

			// FileStorage
			fs, err := c.getFileStorage()
			if err != nil {
				logger.Fatal("failed to setup file storage", zap.Error(err))
			}

			// Repository
			repo, _, err := gorm.NewGormRepository(db, hub.New(), logger, false)
			if err != nil {
				logger.Fatal("failed to initialize repository", zap.Error(err))
			}

			// ImageProcessor
			ip := imaging.NewProcessor(provideImageProcessorConfig(&c))

			// FileManager
			fm, err := file.InitFileManager(repo, fs, ip, logger)
			if err != nil {
				logger.Fatal("failed to initialize file manager", zap.Error(err))
			}

			var groups []*model.UserGroup
			if err := db.Model(&model.UserGroup{}).Where("icon IS NULL").Scan(&groups).Error; err != nil {
				logger.Fatal("failed to get groups", zap.Error(err))
			}

			logger.Info(fmt.Sprintf("Generating default images for %v group(s)", len(groups)))

			for _, group := range groups {
				iconFileID, err := file.GenerateIconFile(fm, group.Name)
				if err != nil {
					logger.Fatal("failed to generate image", zap.Stringer("gid", group.ID), zap.String("group", group.Name), zap.Error(err))
				}
				if err := repo.UpdateUserGroup(group.ID, repository.UpdateUserGroupArgs{Icon: optional.From(iconFileID)}); err != nil {
					logger.Fatal("failed to update user group", zap.Stringer("gid", group.ID), zap.String("group", group.Name), zap.Error(err))
				}
			}

			logger.Info(fmt.Sprintf("Successfully generated images for %v group(s)", len(groups)))
		},
	}
}
