package cmd

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/gormzap"
	"go.uber.org/zap"
	"regexp"
	"strings"
	"time"
)

// migrateV2ToV3Command traQv2データをv3データに変換するコマンド
func migrateV2ToV3Command() *cobra.Command {
	var (
		dryRun bool
	)

	cmd := cobra.Command{
		Use:   "migrate-v2-to-v3",
		Short: "migrate from v2 to v (messages, files)",
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

			// バックアップテーブル作成
			backupTable := fmt.Sprintf("v2_message_backup-%d", time.Now().Unix())
			if !dryRun {
				if err := db.Exec(fmt.Sprintf("CREATE TABLE `%s` (`id` char(36) NOT NULL, `text` text CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL, PRIMARY KEY (`id`)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4", backupTable)).Error; err != nil {
					logger.Fatal(err.Error())
				}
			}

			err = db.Transaction(func(tx *gorm.DB) error {
				var embRegex = regexp.MustCompile(`(?m)!({(?:[ \t\n]*"(?:[^"]|\\.)*"[ \t\n]*:[ \t\n]*"(?:[^"]|\\.)*",)*(?:[ \t\n]*"(?:[^"]|\\.)*"[ \t\n]*:[ \t\n]*"(?:[^"]|\\.)*")})`)
				type embeddedInfo struct {
					Type string    `json:"type"`
					ID   uuid.UUID `json:"id"`
				}
				type fileInfo struct {
					MessageID      uuid.UUID
					ChannelID      uuid.UUID
					MessageDeleted bool
				}

				fileMap := map[uuid.UUID][]*fileInfo{}

				// 全メッセージを1000件ずつ処理
				for page := 0; ; page++ {
					var messages []*model.Message
					if err := tx.
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

					for _, m := range messages {
						var links []string
						converted := embRegex.ReplaceAllStringFunc(m.Text, func(s string) string {
							var info embeddedInfo
							if err := jsoniter.ConfigFastest.Unmarshal([]byte(s[1:]), &info); err != nil {
								return s
							}
							switch info.Type {
							case "file":
								if fs, ok := fileMap[info.ID]; !ok {
									fileMap[info.ID] = []*fileInfo{{
										MessageID:      m.ID,
										ChannelID:      m.ChannelID,
										MessageDeleted: m.DeletedAt != nil,
									}}
								} else {
									fileMap[info.ID] = append(fs, &fileInfo{
										MessageID:      m.ID,
										ChannelID:      m.ChannelID,
										MessageDeleted: m.DeletedAt != nil,
									})
								}
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
							continue // 変化無し
						}

						converted += "\n" + strings.Join(links, "\n")

						if !dryRun {
							// バックアップ
							if err := tx.Exec(fmt.Sprintf("INSERT INTO `%s` VALUES (?, ?)", backupTable), m.ID, m.Text).Error; err != nil {
								return err
							}

							// 書き換え (updated_atは更新しない)
							if err := tx.Model(&m).UpdateColumn("text", converted).Error; err != nil {
								return err
							}
						}
						logger.Info("message: " + m.ID.String())
					}
				}

				for fid, infos := range fileMap {
					var f model.File
					if err := tx.Where("type = '' AND id = ?", fid).First(&f).Error; err != nil {
						if gorm.IsRecordNotFoundError(err) {
							continue
						}
						return err
					}

					// 全部削除されてたら無視
					deleted := true
					for _, info := range infos {
						if !info.MessageDeleted {
							deleted = false
							break
						}
					}
					if deleted {
						continue
					}

					// 異なるチャンネルで貼られているかどうか
					multiple := false
					for _, info := range infos {
						if infos[0].ChannelID != info.ChannelID {
							multiple = true
							break
						}
					}

					if !multiple {
						if !dryRun {
							// 書き換え (updated_atは更新しない)
							if err := tx.Model(&f).UpdateColumn("channel_id", infos[0].ChannelID).Error; err != nil {
								return err
							}
						}
						logger.Info("file: " + fid.String() + " -> " + infos[0].ChannelID.String())
					} else {
						var ids []string
						for _, info := range infos {
							ids = append(ids, info.MessageID.String())
						}
						logger.Warn(fmt.Sprintf("multiple times file attaching detected: %s (%s)", fid, strings.Join(ids, ",")))
					}
				}

				return nil
			})
			if err != nil {
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
	flags.BoolVar(&dryRun, "dry-run", false, "list target files only (no delete)")

	return &cmd
}
