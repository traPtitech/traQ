package cmd

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"math/rand"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/repository/gorm"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/imaging"
	"github.com/traPtitech/traQ/service/rbac/role"
	utilsimaging "github.com/traPtitech/traQ/utils/imaging"
	"github.com/traPtitech/traQ/utils/optional"
	"gorm.io/gorm/clause"
)

const (
	seedUserCount       = 1000
	seedDeactivateCount = 300
	seedChannelCount    = 5000
	seedMessagesPerUser = 100
	seedStampsPerUser   = 10
	messageBatchSize    = 5000
)

// チャンネル名に使う単語リスト ([a-zA-Z0-9-_]{1,20})
var seedChannelWords = []string{
	"general", "random", "announce", "dev", "ops",
	"infra", "backend", "frontend", "design", "ux",
	"mobile", "web", "api", "database", "cloud",
	"security", "testing", "ci-cd", "deploy", "docs",
	"help", "support", "feedback", "project", "team",
	"work", "office", "lunch", "hobby", "game",
	"music", "book", "movie", "news", "tech",
	"blog", "share", "idea", "meeting", "standup",
	"review", "sprint", "kanban", "log", "monitor",
	"alert", "incident", "report", "social", "fun",
	"chat", "talk", "discuss", "bot", "tool",
	"script", "build", "release", "prod", "staging",
	"data", "ml", "ai", "research", "study",
	"recruit", "onboard", "intern", "admin", "sales",
	"marketing", "event", "conference", "workshop", "hack",
	"ios", "android", "linux", "windows", "docker",
	"k8s", "network", "storage", "cache", "queue",
	"auth", "payment", "search", "analytics", "metrics",
	"notification", "webhook", "migration", "backup",
}

// 文頭・本文・文末のパーツを組み合わせて約100文字の日本語メッセージを生成する
var (
	msgOpenings = []string{
		"本日の作業報告です。",
		"進捗を共有します。",
		"確認事項があります。",
		"お疲れさまです。",
		"ご報告があります。",
		"質問があります。",
		"アップデートがあります。",
		"ちょっと相談があるのですが、",
		"先ほどの件について、",
		"調査結果をまとめました。",
		"作業が完了しました。",
		"少しお時間をいただけますか。",
		"フィードバックをいただきたいです。",
		"ミーティングの議事録です。",
	}
	msgBodies = []string{
		"バックエンドAPIのバリデーション処理を修正し、エラーレスポンスの形式を統一しました。",
		"フロントエンドのコンポーネントをリファクタリングして、再利用可能な形に整理しました。",
		"データベースのインデックスを見直し、遅延が発生していたクエリを最適化しました。",
		"CIのパイプラインを改善し、テスト実行時間を従来の半分以下に短縮しました。",
		"ドキュメントを更新しました。APIの仕様変更に合わせて記述を追記・修正しています。",
		"コードレビューのコメントに対応しました。ご指摘の点をすべて修正しています。",
		"ステージング環境へのデプロイが完了し、動作確認が取れています。",
		"依存パッケージのバージョンをアップデートし、脆弱性の対応を行いました。",
		"ログの出力形式を整理して、障害時の調査がしやすい構造に変更しました。",
		"単体テストのカバレッジを80%以上に引き上げました。",
		"パフォーマンステストを実施し、レスポンスタイムが要件を満たしていることを確認しました。",
		"セキュリティレビューの指摘事項に対応し、入力値のサニタイズ処理を追加しました。",
		"マイグレーションスクリプトを作成し、本番環境への適用手順書も用意しました。",
		"モバイルアプリのUIを調整し、デザインガイドラインに沿った表示になりました。",
		"キャッシュ戦略を見直し、不要なAPI呼び出しを削減しました。",
		"新機能のフラグ管理をFeatureToggleで実装し、段階的なリリースが可能になりました。",
		"WebSocketの接続管理を改善し、切断時の再接続処理を安定させました。",
		"認証フローを整理し、トークンのリフレッシュ処理をより堅牢に実装しました。",
		"検索機能のインデックス更新バッチを修正し、データの同期遅延を解消しました。",
		"エラー監視ツールのアラート設定を見直し、誤検知を減らしました。",
		"スプリントの振り返りを行い、次スプリントの改善点を3つ挙げました。",
		"今週のリリース内容を確認しました。特に問題はなさそうです。",
		"ユーザーからのフィードバックを集計し、改善要望の優先度付けをしました。",
		"インフラのコスト最適化を検討した結果、月額で約15%削減できる見込みです。",
		"社内勉強会の資料を作成しました。来週共有させていただきます。",
	}
	msgClosings = []string{
		"ご確認よろしくお願いします。",
		"レビューをお願いします。",
		"ご意見があればお知らせください。",
		"問題があれば教えてください。",
		"引き続きよろしくお願いします。",
		"詳細はPRをご参照ください。",
		"何かあればいつでも声をかけてください。",
		"明日も引き続き作業を進めます。",
		"完了したらまた報告します。",
		"どうぞよろしくお願いします。",
	}
)

// generateStampFile スタンプ用のランダム画像ファイルを生成して保存する
func generateStampFile(ctx context.Context, fm file.Manager, salt string) (uuid.UUID, error) {
	icon, err := utilsimaging.GenerateIcon(salt)
	if err != nil {
		return uuid.Nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, icon); err != nil {
		return uuid.Nil, err
	}
	f, err := fm.Save(ctx, file.SaveArgs{
		FileName:  salt + ".png",
		FileSize:  int64(buf.Len()),
		MimeType:  "image/png",
		FileType:  model.FileTypeStamp,
		Src:       bytes.NewReader(buf.Bytes()),
		Thumbnail: icon,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return f.GetID(), nil
}

func generateSeedMessage() string {
	o := msgOpenings[rand.Intn(len(msgOpenings))]
	b := msgBodies[rand.Intn(len(msgBodies))]
	cl := msgClosings[rand.Intn(len(msgClosings))]
	return o + b + cl
}

// seedCommand ユーザー・チャンネル・メッセージを一括投入するコマンド
func seedCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "seed",
		Short: "Seed users, channels, and messages all at once",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx := context.Background()
			logger, gormLogger := getCLILoggers()
			defer logger.Sync()

			// ── 共通セットアップ ──────────────────────────────────────
			db, err := c.getDatabase()
			if err != nil {
				return fmt.Errorf("failed to connect database: %w", err)
			}
			db.Logger = gormLogger
			sqlDB, err := db.DB()
			if err != nil {
				return fmt.Errorf("failed to get *sql.DB: %w", err)
			}
			defer sqlDB.Close()

			fs, err := c.getFileStorage()
			if err != nil {
				return fmt.Errorf("failed to setup file storage: %w", err)
			}

			repo, _, err := gorm.NewGormRepository(db, hub.New(), logger, false)
			if err != nil {
				return fmt.Errorf("failed to initialize repository: %w", err)
			}

			fm, err := file.InitFileManager(repo, fs, imaging.NewProcessor(provideImageProcessorConfig(&c)), logger)
			if err != nil {
				return fmt.Errorf("failed to initialize file manager: %w", err)
			}

			// システムユーザーロール投入
			if err := repo.CreateUserRoles(ctx, role.SystemRoleModels()...); err != nil {
				logger.Warn("failed to create system roles (may already exist)", zap.Error(err))
			}

			// 管理者ユーザー traq/traq の作成
			adminIconID, err := file.GenerateIconFile(ctx, fm, "traq")
			if err != nil {
				return fmt.Errorf("failed to generate admin icon: %w", err)
			}
			if u, err := repo.CreateUser(ctx, repository.CreateUserArgs{
				Name:       "traq",
				Password:   "traq",
				Role:       role.Admin,
				IconFileID: adminIconID,
			}); err != nil {
				logger.Warn("failed to create traq admin user (may already exist)", zap.Error(err))
			} else {
				logger.Info("traq admin user was created", zap.Stringer("uid", u.GetID()))
			}

			cm, err := channel.InitChannelManager(repo, logger)
			if err != nil {
				return fmt.Errorf("failed to initialize channel manager: %w", err)
			}

			// #general チャンネル作成
			if ch, err := cm.CreatePublicChannel(ctx, "general", uuid.Nil, uuid.Nil); err != nil {
				logger.Warn("failed to create #general (may already exist)", zap.Error(err))
			} else {
				logger.Info("#general was created", zap.Stringer("cid", ch.ID))
			}

			// ── フェーズ1: ユーザー作成 ───────────────────────────────
			logger.Info("=== phase 1: users ===")
			allUserIDs := make([]uuid.UUID, 0, seedUserCount)

			for i := range seedUserCount {
				name := fmt.Sprintf("seed_user_%04d", i+1)
				iconFileID, err := file.GenerateIconFile(ctx, fm, name)
				if err != nil {
					return fmt.Errorf("failed to generate icon for %s: %w", name, err)
				}
				u, err := repo.CreateUser(ctx, repository.CreateUserArgs{
					Name:       name,
					Password:   "P@ssw0rd",
					Role:       role.User,
					IconFileID: iconFileID,
				})
				if err != nil {
					return fmt.Errorf("failed to create user %s: %w", name, err)
				}
				allUserIDs = append(allUserIDs, u.GetID())
				if (i+1)%100 == 0 {
					logger.Info(fmt.Sprintf("  created %d / %d users", i+1, seedUserCount))
				}
			}

			perm := rand.Perm(seedUserCount)
			for _, idx := range perm[:seedDeactivateCount] {
				if err := repo.UpdateUser(ctx, allUserIDs[idx], repository.UpdateUserArgs{
					UserState: optional.From(model.UserAccountStatusDeactivated),
				}); err != nil {
					return fmt.Errorf("failed to deactivate user (index %d): %w", idx, err)
				}
			}
			deactivatedSet := make(map[uuid.UUID]bool, seedDeactivateCount)
			for _, idx := range perm[:seedDeactivateCount] {
				deactivatedSet[allUserIDs[idx]] = true
			}

			// アクティブユーザーIDリスト（チャンネル作成用）
			activeUserIDs := make([]uuid.UUID, 0, seedUserCount-seedDeactivateCount)
			for _, uid := range allUserIDs {
				if !deactivatedSet[uid] {
					activeUserIDs = append(activeUserIDs, uid)
				}
			}

			logger.Info("phase 1 done",
				zap.Int("created", seedUserCount),
				zap.Int("deactivated", seedDeactivateCount),
			)

			// ── フェーズ2: チャンネル作成 ─────────────────────────────
			logger.Info("=== phase 2: channels ===")

			const maxParentDepth = 3
			byDepth := make([][]uuid.UUID, maxParentDepth+2) // [0..4]
			for i := range byDepth {
				byDepth[i] = make([]uuid.UUID, 0, 64)
			}
			depthProb := []float64{0.03, 0.10, 0.22, 0.33, 0.32}
			usedNames := map[uuid.UUID]map[string]bool{uuid.Nil: {}}

			pickParent := func() (uuid.UUID, int) {
				totalWeight := 0.0
				weights := make([]float64, len(depthProb))
				for tier, w := range depthProb {
					if tier == 0 || len(byDepth[tier-1]) > 0 {
						weights[tier] = w
						totalWeight += w
					}
				}
				r := rand.Float64() * totalWeight
				cum := 0.0
				for tier, w := range weights {
					cum += w
					if r <= cum {
						if tier == 0 {
							return uuid.Nil, 0
						}
						pool := byDepth[tier-1]
						return pool[rand.Intn(len(pool))], tier
					}
				}
				return uuid.Nil, 0
			}

			pickName := func(parentID uuid.UUID) string {
				used, ok := usedNames[parentID]
				if !ok {
					used = map[string]bool{}
					usedNames[parentID] = used
				}
				for _, idx := range rand.Perm(len(seedChannelWords)) {
					name := seedChannelWords[idx]
					if !used[name] {
						used[name] = true
						return name
					}
				}
				for n := 2; ; n++ {
					name := fmt.Sprintf("ch-%d", n)
					if !used[name] {
						used[name] = true
						return name
					}
				}
			}

			allChannelIDs := make([]uuid.UUID, 0, seedChannelCount)
			for i := range seedChannelCount {
				parentID, tier := pickParent()
				name := pickName(parentID)
				creator := activeUserIDs[rand.Intn(len(activeUserIDs))]

				ch, err := cm.CreatePublicChannel(ctx, name, parentID, creator)
				if err == channel.ErrChannelNameConflicts {
					name = pickName(parentID)
					ch, err = cm.CreatePublicChannel(ctx, name, parentID, creator)
				}
				if err != nil {
					return fmt.Errorf("failed to create channel (i=%d): %w", i, err)
				}

				childDepth := tier
				byDepth[childDepth] = append(byDepth[childDepth], ch.ID)
				usedNames[ch.ID] = map[string]bool{}
				allChannelIDs = append(allChannelIDs, ch.ID)

				if (i+1)%500 == 0 {
					logger.Info(fmt.Sprintf("  created %d / %d channels", i+1, seedChannelCount))
				}
			}

			dist := make([]int, len(byDepth))
			for d, pool := range byDepth {
				dist[d] = len(pool)
			}
			logger.Info("phase 2 done",
				zap.Int("total", seedChannelCount),
				zap.Ints("by_depth", dist),
			)

			// ── フェーズ3: メッセージ投稿（バッチ挿入） ──────────────────
			logger.Info("=== phase 3: messages ===")

			totalMessages := len(activeUserIDs) * seedMessagesPerUser
			logger.Info(fmt.Sprintf("creating %d messages (%d users × %d msgs)...",
				totalMessages, len(activeUserIDs), seedMessagesPerUser))

			// 20%のメッセージにスタンプを付ける用にIDを控える
			stampedMsgIDs := make([]uuid.UUID, 0, totalMessages/5)
			batch := make([]model.Message, 0, messageBatchSize)
			created := 0
			for _, uid := range activeUserIDs {
				for range seedMessagesPerUser {
					msgID := uuid.Must(uuid.NewV7())
					if rand.Float64() < 0.2 {
						stampedMsgIDs = append(stampedMsgIDs, msgID)
					}
					batch = append(batch, model.Message{
						ID:        msgID,
						UserID:    uid,
						ChannelID: allChannelIDs[rand.Intn(len(allChannelIDs))],
						Text:      generateSeedMessage(),
					})
					if len(batch) >= messageBatchSize {
						if err := db.Omit(clause.Associations).Create(&batch).Error; err != nil {
							return fmt.Errorf("failed to batch insert messages: %w", err)
						}
						created += len(batch)
						batch = batch[:0]
						if created%100000 == 0 {
							logger.Info(fmt.Sprintf("  created %d / %d messages", created, totalMessages))
						}
					}
				}
			}
			if len(batch) > 0 {
				if err := db.Omit(clause.Associations).Create(&batch).Error; err != nil {
					return fmt.Errorf("failed to batch insert messages: %w", err)
				}
				created += len(batch)
			}
			logger.Info(fmt.Sprintf("  %d messages marked for stamp reactions", len(stampedMsgIDs)))

			// ChannelLatestMessage を一括更新
			logger.Info("updating channel latest messages...")
			if err := db.Exec(`
				INSERT INTO channel_latest_messages (channel_id, message_id, date_time)
				SELECT m.channel_id, m.id, m.created_at
				FROM messages m
				INNER JOIN (
					SELECT channel_id, MAX(created_at) AS max_created
					FROM messages WHERE deleted_at IS NULL GROUP BY channel_id
				) latest ON m.channel_id = latest.channel_id AND m.created_at = latest.max_created
				WHERE m.deleted_at IS NULL
				ON DUPLICATE KEY UPDATE message_id = VALUES(message_id), date_time = VALUES(date_time)
			`).Error; err != nil {
				return fmt.Errorf("failed to update channel latest messages: %w", err)
			}

			logger.Info("phase 3 done", zap.Int("total", created))

			// ── フェーズ4: スタンプ作成 ───────────────────────────────
			logger.Info("=== phase 4: stamps ===")

			allStampIDs := make([]uuid.UUID, 0, seedUserCount*seedStampsPerUser)
			for i, uid := range allUserIDs {
				for j := range seedStampsPerUser {
					salt := fmt.Sprintf("seed-stamp-%d-%d", i+1, j+1)
					fileID, err := generateStampFile(ctx, fm, salt)
					if err != nil {
						return fmt.Errorf("failed to generate stamp file (user=%d, stamp=%d): %w", i+1, j+1, err)
					}
					s, err := repo.CreateStamp(ctx, repository.CreateStampArgs{
						Name:      fmt.Sprintf("s%d-%d", i+1, j+1),
						FileID:    fileID,
						CreatorID: uid,
					})
					if err != nil {
						return fmt.Errorf("failed to create stamp (user=%d, stamp=%d): %w", i+1, j+1, err)
					}
					allStampIDs = append(allStampIDs, s.ID)
				}
				if (i+1)%100 == 0 {
					logger.Info(fmt.Sprintf("  stamped %d / %d users", i+1, seedUserCount))
				}
			}

			logger.Info("phase 4 done", zap.Int("total", len(allStampIDs)))

			// ── フェーズ5: メッセージスタンプ（リアクション）─────────────
			logger.Info("=== phase 5: message stamp reactions ===")
			logger.Info(fmt.Sprintf("adding reactions to %d messages...", len(stampedMsgIDs)))

			const reactionBatchSize = 10000
			reactionBatch := make([]model.MessageStamp, 0, reactionBatchSize)
			reactionTotal := 0
			for _, msgID := range stampedMsgIDs {
				numReactors := 8 + rand.Intn(5) // 8〜12人
				perm := rand.Perm(len(activeUserIDs))
				for _, ri := range perm[:numReactors] {
					reactionBatch = append(reactionBatch, model.MessageStamp{
						MessageID: msgID,
						StampID:   allStampIDs[rand.Intn(len(allStampIDs))],
						UserID:    activeUserIDs[ri],
						Count:     1,
					})
					if len(reactionBatch) >= reactionBatchSize {
						if err := db.Omit(clause.Associations).Create(&reactionBatch).Error; err != nil {
							return fmt.Errorf("failed to batch insert reactions: %w", err)
						}
						reactionTotal += len(reactionBatch)
						reactionBatch = reactionBatch[:0]
						if reactionTotal%500000 == 0 {
							logger.Info(fmt.Sprintf("  created %d reactions", reactionTotal))
						}
					}
				}
			}
			if len(reactionBatch) > 0 {
				if err := db.Omit(clause.Associations).Create(&reactionBatch).Error; err != nil {
					return fmt.Errorf("failed to batch insert reactions: %w", err)
				}
				reactionTotal += len(reactionBatch)
			}

			logger.Info("phase 5 done", zap.Int("total", reactionTotal))
			logger.Info("=== seed all completed ===",
				zap.Int("users", seedUserCount),
				zap.Int("channels", seedChannelCount),
				zap.Int("messages", totalMessages),
				zap.Int("stamps", len(allStampIDs)),
				zap.Int("reactions", reactionTotal),
			)
			return nil
		},
	}
}
