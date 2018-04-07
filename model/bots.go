package model

import (
	"encoding/base64"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/bot"
	"github.com/traPtitech/traQ/bot/events"
	"github.com/traPtitech/traQ/rbac/role"
	"sync"
	"time"
)

// BotStoreImpl Botデータ用ストアの実装
type BotStoreImpl struct {
	webhooks    sync.Map
	plugins     sync.Map
	generalBots sync.Map
}

// Init BotStoreを初期化。必ず呼ぶこと
func (s *BotStoreImpl) Init() error {
	var webhooks []*WebhookBotUser
	err := db.Join("INNER", "webhook_bots", "webhook_bots.bot_user_id = users.id").Find(&webhooks)
	if err != nil {
		return err
	}
	for _, v := range webhooks {
		id := uuid.Must(uuid.FromString(v.WebhookBot.ID))
		s.webhooks.Store(id, bot.Webhook{
			ID:          id,
			BotUserID:   uuid.Must(uuid.FromString(v.WebhookBot.BotUserID)),
			Name:        v.User.DisplayName,
			Description: v.WebhookBot.Description,
			ChannelID:   uuid.Must(uuid.FromString(v.WebhookBot.ChannelID)),
			IconFileID:  uuid.Must(uuid.FromString(v.User.Icon)),
			CreatorID:   uuid.Must(uuid.FromString(v.WebhookBot.CreatorID)),
			CreatedAt:   v.WebhookBot.CreatedAt,
			UpdatedAt:   v.WebhookBot.UpdatedAt,
			IsValid:     v.WebhookBot.IsValid,
		})
	}
	return nil
}

// SaveWebhook Webhookをdbに保存
func (s *BotStoreImpl) SaveWebhook(w *bot.Webhook) error {
	_, ok := s.GetWebhook(w.ID)
	if !ok {
		u := &User{
			ID:          w.BotUserID.String(),
			Name:        "Webhook#" + base64.RawStdEncoding.EncodeToString(w.BotUserID.Bytes()),
			DisplayName: w.Name,
			Email:       "",
			Password:    "",
			Salt:        "",
			Icon:        w.IconFileID.String(),
			Status:      0, //TODO
			Bot:         true,
			BotType:     bot.TypeWebhook,
			Role:        role.Bot.ID(), //FIXME
		}
		wb := &WebhookBot{
			ID:          w.ID.String(),
			BotUserID:   w.BotUserID.String(),
			Description: w.Description,
			ChannelID:   w.ChannelID.String(),
			IsValid:     w.IsValid,
			CreatorID:   w.CreatorID.String(),
			CreatedAt:   w.CreatedAt,
			UpdatedAt:   w.UpdatedAt,
		}

		if _, err := db.Insert(u, wb); err != nil {
			return err
		}
		s.webhooks.Store(w.ID, *w)
	} else {
		if _, err := db.ID(w.BotUserID.String()).Update(&User{
			DisplayName: w.Name,
			Icon:        w.IconFileID.String(),
		}); err != nil {
			return err
		}
		if _, err := db.ID(w.ID.String()).UseBool("is_valid").Update(&WebhookBot{
			Description: w.Description,
			ChannelID:   w.ChannelID.String(),
			IsValid:     w.IsValid,
			CreatorID:   w.CreatorID.String(),
			UpdatedAt:   w.UpdatedAt,
		}); err != nil {
			return err
		}
		s.webhooks.Store(w.ID, *w)
	}

	return nil
}

// GetAllWebhooks Webhookを全て取得
func (s *BotStoreImpl) GetAllWebhooks() (arr []bot.Webhook) {
	s.webhooks.Range(func(key, value interface{}) bool {
		arr = append(arr, value.(bot.Webhook))
		return true
	})
	return
}

// GetWebhook Webhookを取得
func (s *BotStoreImpl) GetWebhook(id uuid.UUID) (bot.Webhook, bool) {
	w, ok := s.webhooks.Load(id)
	if !ok {
		return bot.Webhook{}, false
	}
	return w.(bot.Webhook), true
}

// Plugin Plugin構造体
type Plugin struct {
	ID                string    `xorm:"char(36) not null pk"`
	BotUserID         string    `xorm:"char(36) not null unique"`
	Command           string    `xorm:"char(50) not null unique"`
	Description       string    `xorm:"text not null"`
	Usage             string    `xorm:"text not null"`
	VerificationToken string    `xorm:"text not null"`
	AccessTokenID     string    `xorm:"char(36) not null"`
	PostURL           string    `xorm:"text not null"`
	Tested            bool      `xorm:"bool not null"`
	IsValid           bool      `xorm:"bool not null"`
	CreatorID         string    `xorm:"char(36) not null"`
	CreatedAt         time.Time `xorm:"timestamp not null"`
	UpdatedAt         time.Time `xorm:"timestamp not null"`
}

// WebhookBot WebhookBot構造体
type WebhookBot struct {
	ID          string    `xorm:"char(36) not null pk"`
	BotUserID   string    `xorm:"char(36) not null unique"`
	Description string    `xorm:"text not null"`
	ChannelID   string    `xorm:"char(36) not null"`
	IsValid     bool      `xorm:"bool not null"`
	CreatorID   string    `xorm:"char(36) not null"`
	CreatedAt   time.Time `xorm:"timestamp not null"`
	UpdatedAt   time.Time `xorm:"timestamp not null"`
}

// GeneralBot GeneralBot構造体
type GeneralBot struct {
	ID                    string                    `xorm:"char(36) not null pk"`
	BotUserID             string                    `xorm:"char(36) not null unique"`
	Description           string                    `xorm:"text not null"`
	VerificationToken     string                    `xorm:"char(36) not null"`
	AccessTokenID         string                    `xorm:"char(36) not null"`
	PostURL               string                    `xorm:"text not null"`
	HookedEvents          string                    `xorm:"text not null"`
	Tested                bool                      `xorm:"bool not null"`
	parsedHookedEventsRaw string                    `xorm:"-"`
	parsedHookedEvents    map[events.EventType]bool `xorm:"-"`
	IsValid               bool                      `xorm:"bool not null"`
	CreatorID             string                    `xorm:"char(36) not null"`
	CreatedAt             time.Time                 `xorm:"timestamp not null"`
	UpdatedAt             time.Time                 `xorm:"timestamp not null"`
}

// BotOutgoingPostLog BotのPOST URLへのリクエストの結果のログ構造体
type BotOutgoingPostLog struct {
	ID         string    `xorm:"char(36) not null pk"`
	BotUserID  string    `xorm:"char(36) not null"`
	StatusCode int       `xorm:"int not null"`
	Summary    string    `xorm:"text not null"`
	Datetime   time.Time `xorm:"created"`
}

// TableName Pluginのテーブル名
func (*Plugin) TableName() string {
	return "plugins"
}

// TableName Webhookのテーブル名
func (*WebhookBot) TableName() string {
	return "webhook_bots"
}

// TableName GeneralBotのテーブル名
func (*GeneralBot) TableName() string {
	return "general_bots"
}

// TableName BotOutgoingPostLogのテーブル名
func (*BotOutgoingPostLog) TableName() string {
	return "bot_outgoing_post_logs"
}

// WebhookBotUser WebhookBotUser構造体 内部にUser, WebhookBotを内包
type WebhookBotUser struct {
	*User       `xorm:"extends"`
	*WebhookBot `xorm:"extends"`
}

// TableName JOIN処理用
func (*WebhookBotUser) TableName() string {
	return "users"
}
