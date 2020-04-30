package cmd

import (
	"context"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/gormzap"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"regexp"
	"strings"
	"sync"
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

			// バックアップ・作業テーブル作成
			if err := db.AutoMigrate(&V2MessageBackup{}, &V2MessageFileMapping{}).Error; err != nil {
				logger.Fatal(err.Error())
			}

			// メッセージ変換
			if !skipConvertMessage {
				if err := convertMessages(db, logger, dryRun, startMessagePage); err != nil {
					logger.Fatal(err.Error())
				}
			}

			// ファイルのチャンネル紐付け
			if err := linkFileToChannel(db, logger, dryRun, startFilePage); err != nil {
				logger.Fatal(err.Error())
			}
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
	flags.String("origin", "", "traQ origin")
	bindPFlag(flags, "origin", "origin")
	flags.BoolVar(&dryRun, "dry-run", false, "dry run")
	flags.BoolVar(&skipConvertMessage, "skip-convert-message", false, "skip message converting")
	flags.IntVar(&startMessagePage, "start-message-page", 0, "start message page (zero-origin)")
	flags.IntVar(&startFilePage, "start-file-page", 0, "start file page (zero-origin)")

	return &cmd
}

type V2MessageBackup struct {
	MessageID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	OldText   string    `gorm:"type:text;not null"`
	NewText   string    `gorm:"type:text;not null"`
}

type V2MessageFileMapping struct {
	FileID         uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	MessageID      uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	ChannelID      uuid.UUID `gorm:"type:char(36);not null"`
	MessageDeleted bool      `gorm:"type:boolean;not null"`
}

func convertMessages(db *gorm.DB, logger *zap.Logger, dryRun bool, startMessagePage int) error {
	embRegex := regexp.MustCompile(`(?m)!({(?:[ \t\n]*"(?:[^"]|\\.)*"[ \t\n]*:[ \t\n]*"(?:[^"]|\\.)*",)*(?:[ \t\n]*"(?:[^"]|\\.)*"[ \t\n]*:[ \t\n]*"(?:[^"]|\\.)*")})`)
	for page := startMessagePage; ; page++ {
		logger.Info(fmt.Sprintf("messages_page: %d", page))

		var messages []*model.Message
		if err := db.
			Unscoped(). // Unscopedで削除メッセージも取得
			Limit(1000).
			Offset(page * 1000).
			Order("created_at ASC").
			Find(&messages).Error; err != nil {
			return err
		}
		if len(messages) == 0 {
			break
		}

		var fail bool
		var failLock sync.Mutex

		wg := &sync.WaitGroup{}
		s := semaphore.NewWeighted(10)
		for _, m := range messages {
			wg.Add(1)
			go func(m *model.Message) {
				defer wg.Done()

				var links []string
				var files []*V2MessageFileMapping
				converted := embRegex.ReplaceAllStringFunc(m.Text, func(s string) string {
					var info struct {
						Type string    `json:"type"`
						ID   uuid.UUID `json:"id"`
					}
					if err := jsoniter.ConfigFastest.Unmarshal([]byte(s[1:]), &info); err != nil {
						return s
					}
					switch info.Type {
					case "file":
						files = append(files, &V2MessageFileMapping{
							FileID:         info.ID,
							MessageID:      m.ID,
							ChannelID:      m.ChannelID,
							MessageDeleted: m.DeletedAt != nil,
						})
						links = append(links, c.Origin+"/files/"+info.ID.String())
						return ""
					case "message":
						links = append(links, c.Origin+"/messages/"+info.ID.String())
						return ""
					default:
						return s
					}
				})
				if len(links) == 0 {
					return // 変化無し
				}
				if len(converted) > 0 && !strings.HasSuffix(converted, "\n") {
					converted += "\n"
				}
				converted += strings.Join(links, "\n")

				if !dryRun {
					s.Acquire(context.Background(), 1)
					defer s.Release(1)

					err := db.Transaction(func(tx *gorm.DB) error {
						// バックアップ
						if err := tx.Create(&V2MessageBackup{
							MessageID: m.ID,
							OldText:   m.Text,
							NewText:   converted,
						}).Error; err != nil {
							return err
						}

						// ファイルマッピング情報保存
						for _, file := range files {
							if err := tx.Create(file).Error; err != nil {
								return err
							}
						}

						// 書き換え (updated_atは更新しない)
						if err := tx.Unscoped().Model(m).UpdateColumn("text", converted).Error; err != nil {
							return err
						}

						return nil
					})
					if err != nil {
						logger.Error(err.Error())
						failLock.Lock()
						fail = true
						failLock.Unlock()
						return
					}
				}
				logger.Info("message: " + m.ID.String())
			}(m)
		}
		wg.Wait()

		if fail {
			logger.Fatal("error occurred")
		}
	}
	return nil
}

func linkFileToChannel(db *gorm.DB, logger *zap.Logger, dryRun bool, startFilePage int) error {
	for page := startFilePage; ; page++ {
		logger.Info(fmt.Sprintf("files_page: %d", page))

		var files []*model.File
		if err := db.
			Where("type = ''").
			Limit(1000).
			Offset(page * 1000).
			Order("id ASC").
			Find(&files).Error; err != nil {
			return err
		}
		if len(files) == 0 {
			break
		}

		var fail bool
		var failLock sync.Mutex

		wg := &sync.WaitGroup{}
		s := semaphore.NewWeighted(10)
		for _, file := range files {
			if file.ChannelID.Valid {
				continue
			}

			wg.Add(1)
			s.Acquire(context.Background(), 1)
			go func(file *model.File) {
				defer wg.Done()
				defer s.Release(1)

				var mappings []V2MessageFileMapping
				if err := db.Where(&V2MessageFileMapping{FileID: file.ID}).Find(&mappings).Error; err != nil {
					logger.Error(err.Error())
					failLock.Lock()
					fail = true
					failLock.Unlock()
					return
				}
				if len(mappings) == 0 {
					return
				}

				// 全部削除されてたら無視
				deleted := true
				for _, mapping := range mappings {
					if !mapping.MessageDeleted {
						deleted = false
						break
					}
				}
				if deleted {
					return
				}

				// 異なるチャンネルで貼られているかどうか
				multiple := false
				for _, mapping := range mappings {
					if mappings[0].ChannelID != mapping.ChannelID {
						multiple = true
						break
					}
				}
				if multiple {
					var ids []string
					for _, mapping := range mappings {
						ids = append(ids, mapping.MessageID.String())
					}
					logger.Warn(fmt.Sprintf("multiple times file attaching detected: %s (%s)", file.ID, strings.Join(ids, ",")))
					return
				}

				if !dryRun {
					// 書き換え (updated_atは更新しない)
					if err := db.Model(&file).UpdateColumn("channel_id", mappings[0].ChannelID).Error; err != nil {
						logger.Error(err.Error())
						failLock.Lock()
						fail = true
						failLock.Unlock()
						return
					}
				}
				logger.Info("file: " + file.ID.String() + " -> " + mappings[0].ChannelID.String())
			}(file)
		}
		wg.Wait()

		if fail {
			logger.Fatal("error occurred")
		}
	}
	return nil
}
