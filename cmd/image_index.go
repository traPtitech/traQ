package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/repository/gorm"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/search"
	"github.com/traPtitech/traQ/utils/optional"
)

const imageIndexBatchSize = 100

// imageIndexCommand 画像インデックス再作成コマンド
func imageIndexCommand() *cobra.Command {
	var clearIndex bool

	cmd := cobra.Command{
		Use:   "image-index",
		Short: "Reindex image OCR/embedding data for Elasticsearch",
		Long: `Process messages with image attachments and update Elasticsearch index
with OCR text and embedding vectors.

Only messages that have not yet been processed are targeted.
Use --clear to remove existing image index data before processing,
which is useful when the embedding model has been changed.`,
		Run: func(_ *cobra.Command, _ []string) {
			logger, gormLogger := getCLILoggers()
			defer logger.Sync()

			logger.Info("starting image index batch")

			// Database
			engine, err := c.getDatabase()
			if err != nil {
				logger.Fatal("failed to connect database", zap.Error(err))
			}
			engine.Logger = gormLogger
			db, err := engine.DB()
			if err != nil {
				logger.Fatal("failed to get *sql.DB", zap.Error(err))
			}
			defer db.Close()

			// FileStorage
			fs, err := c.getFileStorage()
			if err != nil {
				logger.Fatal("failed to setup file storage", zap.Error(err))
			}

			// Repository
			repo, _, err := gorm.NewGormRepository(engine, nil, logger, false)
			if err != nil {
				logger.Fatal("failed to initialize repository", zap.Error(err))
			}

			// Channel Manager
			cm, err := channel.InitChannelManager(repo, logger)
			if err != nil {
				logger.Fatal("failed to initialize channel manager", zap.Error(err))
			}

			// Message Manager
			mm, err := message.NewMessageManager(repo, cm, logger)
			if err != nil {
				logger.Fatal("failed to initialize message manager", zap.Error(err))
			}

			// Search Engine
			esConfig := provideESEngineConfig(&c)
			searchEngine, err := search.NewESEngine(mm, cm, repo, fs, logger, esConfig)
			if err != nil {
				logger.Fatal("failed to initialize search engine", zap.Error(err))
			}
			defer searchEngine.Close()

			ctx := context.Background()

			// --clear: 既存の画像インデックスデータをクリア
			if clearIndex {
				logger.Info("clearing existing image index data")
				if err := searchEngine.ClearImageIndex(ctx); err != nil {
					logger.Fatal("failed to clear image index", zap.Error(err))
				}
			}

			// ESから未処理のメッセージIDを取得
			unprocessedIDs, err := searchEngine.GetUnprocessedImageMessageIDs(ctx)
			if err != nil {
				logger.Fatal("failed to get unprocessed message IDs", zap.Error(err))
			}
			logger.Info(fmt.Sprintf("found %d unprocessed messages with images", len(unprocessedIDs)))

			// バッチごとにDBからメッセージを取得して処理
			totalProcessed := 0
			for i := 0; i < len(unprocessedIDs); i += imageIndexBatchSize {
				end := i + imageIndexBatchSize
				if end > len(unprocessedIDs) {
					end = len(unprocessedIDs)
				}
				batch := unprocessedIDs[i:end]

				messages, _, err := repo.GetMessages(ctx, repository.MessagesQuery{
					IDIn:           optional.From(batch),
					Limit:          len(batch),
					DisablePreload: true,
				})
				if err != nil {
					logger.Fatal("failed to get messages from DB", zap.Error(err))
				}

				if err := searchEngine.ProcessImagesForMessages(ctx, messages); err != nil {
					logger.Error("failed to process image batch, progress is saved in ES",
						zap.Error(err),
						zap.Int("processed", totalProcessed))
					break
				}

				totalProcessed += len(messages)
				logger.Info(fmt.Sprintf("processed %d/%d messages", totalProcessed, len(unprocessedIDs)))
			}

			logger.Info(fmt.Sprintf("image index batch completed. processed: %d", totalProcessed))
		},
	}

	cmd.Flags().BoolVar(&clearIndex, "clear", false, "Clear existing image index data before processing (use when embedding model has changed)")

	return &cmd
}

// imageIndexClearCommand 画像インデックスデータのみをクリアするコマンド
func imageIndexClearCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "image-index-clear",
		Short: "Clear image OCR/embedding data from Elasticsearch index",
		Long:  "Remove imageText and imageVector fields from all documents. Run image-index afterwards to re-process.",
		Run: func(_ *cobra.Command, _ []string) {
			logger, gormLogger := getCLILoggers()
			defer logger.Sync()

			// Database
			engine, err := c.getDatabase()
			if err != nil {
				logger.Fatal("failed to connect database", zap.Error(err))
			}
			engine.Logger = gormLogger
			db, err := engine.DB()
			if err != nil {
				logger.Fatal("failed to get *sql.DB", zap.Error(err))
			}
			defer db.Close()

			// FileStorage
			fs, err := c.getFileStorage()
			if err != nil {
				logger.Fatal("failed to setup file storage", zap.Error(err))
			}

			// Repository
			repo, _, err := gorm.NewGormRepository(engine, nil, logger, false)
			if err != nil {
				logger.Fatal("failed to initialize repository", zap.Error(err))
			}

			// Channel Manager
			cm, err := channel.InitChannelManager(repo, logger)
			if err != nil {
				logger.Fatal("failed to initialize channel manager", zap.Error(err))
			}

			// Message Manager
			mm, err := message.NewMessageManager(repo, cm, logger)
			if err != nil {
				logger.Fatal("failed to initialize message manager", zap.Error(err))
			}

			// Search Engine
			esConfig := provideESEngineConfig(&c)
			searchEngine, err := search.NewESEngine(mm, cm, repo, fs, logger, esConfig)
			if err != nil {
				logger.Fatal("failed to initialize search engine", zap.Error(err))
			}
			defer searchEngine.Close()

			ctx := context.Background()
			if err := searchEngine.ClearImageIndex(ctx); err != nil {
				logger.Fatal("failed to clear image index", zap.Error(err))
			}
			logger.Info("image index data cleared successfully")
		},
	}

	return &cmd
}
