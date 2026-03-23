package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/repository/gorm"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/search"
)

const imageIndexBatchSize = 100

// imageIndexCommand 画像インデックス再作成コマンド
func imageIndexCommand() *cobra.Command {
	cmd := cobra.Command{
		Use:   "image-index",
		Short: "Reindex image OCR/embedding data for Elasticsearch",
		Long:  "Process all messages with image attachments and update Elasticsearch index with OCR text and embedding vectors.",
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

			// 全メッセージを走査して画像付きメッセージを処理
			ctx := context.Background()
			lastUpdated := time.Time{}
			totalProcessed := 0

			for {
				messages, more, err := repo.GetUpdatedMessagesAfter(ctx, lastUpdated, imageIndexBatchSize)
				if err != nil {
					logger.Fatal("failed to get messages", zap.Error(err))
				}
				if len(messages) == 0 {
					break
				}
				lastUpdated = messages[len(messages)-1].UpdatedAt

				if err := searchEngine.ProcessImagesForMessages(ctx, messages); err != nil {
					logger.Error("failed to process image batch", zap.Error(err))
					break
				}

				totalProcessed += len(messages)
				logger.Info(fmt.Sprintf("processed %d messages so far", totalProcessed))

				if !more {
					break
				}
			}

			logger.Info(fmt.Sprintf("image index batch completed. total messages scanned: %d", totalProcessed))
		},
	}

	return &cmd
}
