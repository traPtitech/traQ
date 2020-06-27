package v2tov3

import (
	"context"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	jsoniter "github.com/json-iterator/go"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/gormutil"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"regexp"
	"strings"
	"sync"
)

func Run(db *gorm.DB, logger *zap.Logger, origin string, dryRun bool, startMessagePage int, startFilePage int, skipConvertMessage bool) error {
	// バックアップ・作業テーブル作成
	if err := db.AutoMigrate(&V2MessageBackup{}, &V2MessageFileMapping{}).Error; err != nil {
		return err
	}

	// メッセージ変換
	if !skipConvertMessage {
		if err := convertMessages(db, logger, origin, dryRun, startMessagePage); err != nil {
			return err
		}
	}

	// ファイルのチャンネル紐付け
	if err := linkFileToChannel(db, logger, dryRun, startFilePage); err != nil {
		return err
	}
	return nil
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

func convertMessages(db *gorm.DB, logger *zap.Logger, origin string, dryRun bool, startMessagePage int) error {
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
						links = append(links, origin+"/files/"+info.ID.String())
						return ""
					case "message":
						links = append(links, origin+"/messages/"+info.ID.String())
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
					_ = s.Acquire(context.Background(), 1)
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
								if gormutil.IsMySQLDuplicatedRecordErr(err) {
									continue
								}
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

		var files []*model.FileMeta
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
			_ = s.Acquire(context.Background(), 1)
			go func(file *model.FileMeta) {
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
