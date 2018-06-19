package model

import (
	"encoding/base64"
	"errors"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/rbac/role"
	"time"
	"unicode/utf8"
)

// Webhook Webhook
type Webhook interface {
	ID() uuid.UUID
	BotUserID() uuid.UUID
	Name() string
	Description() string
	ChannelID() uuid.UUID
	CreatorID() uuid.UUID
	CreatedAt() time.Time
	UpdatedAt() time.Time
}

// WebhookBot DB用WebhookBot構造体
type WebhookBot struct {
	ID          string     `xorm:"char(36) not null pk"`
	BotUserID   string     `xorm:"char(36) not null unique"`
	Description string     `xorm:"text not null"`
	ChannelID   string     `xorm:"char(36) not null"`
	CreatorID   string     `xorm:"char(36) not null"`
	CreatedAt   time.Time  `xorm:"created not null"`
	UpdatedAt   time.Time  `xorm:"updated not null"`
	DeletedAt   *time.Time `xorm:"timestamp"`
}

// TableName Webhookのテーブル名
func (*WebhookBot) TableName() string {
	return "webhook_bots"
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

// ID WebhookのID
func (w *WebhookBotUser) ID() uuid.UUID {
	return uuid.Must(uuid.FromString(w.WebhookBot.ID))
}

// BotUserID WebhookのUserID
func (w *WebhookBotUser) BotUserID() uuid.UUID {
	return uuid.Must(uuid.FromString(w.WebhookBot.BotUserID))
}

// Name Webhook名
func (w *WebhookBotUser) Name() string {
	return w.User.DisplayName
}

// Description Webhookの説明
func (w *WebhookBotUser) Description() string {
	return w.WebhookBot.Description
}

// ChannelID Webhookの投稿先チャンネルのID
func (w *WebhookBotUser) ChannelID() uuid.UUID {
	return uuid.Must(uuid.FromString(w.WebhookBot.ChannelID))
}

// CreatorID Webhookの作成者
func (w *WebhookBotUser) CreatorID() uuid.UUID {
	return uuid.Must(uuid.FromString(w.WebhookBot.CreatorID))
}

// CreatedAt Webhookの作成日時
func (w *WebhookBotUser) CreatedAt() time.Time {
	return w.WebhookBot.CreatedAt
}

// UpdatedAt Webhookの更新日時
func (w *WebhookBotUser) UpdatedAt() time.Time {
	return w.WebhookBot.UpdatedAt
}

// CreateWebhook Webhookを作成します
func CreateWebhook(name, description string, channelID, creatorID, iconFileID uuid.UUID) (Webhook, error) {
	if len(name) == 0 || utf8.RuneCountInString(name) > 32 {
		return nil, errors.New("invalid name")
	}
	if len(description) == 0 {
		return nil, errors.New("description is required")
	}
	uid := uuid.NewV4()
	bid := uuid.NewV4()

	u := &User{
		ID:          uid.String(),
		Name:        "Webhook#" + base64.RawStdEncoding.EncodeToString(uid.Bytes()),
		DisplayName: name,
		Email:       "",
		Password:    "",
		Salt:        "",
		Icon:        iconFileID.String(),
		Bot:         true,
		Role:        role.Bot.ID(),
	}
	wb := &WebhookBot{
		ID:          bid.String(),
		BotUserID:   uid.String(),
		Description: description,
		ChannelID:   channelID.String(),
		CreatorID:   creatorID.String(),
	}

	_, err := db.UseBool().Insert(u, wb)
	return &WebhookBotUser{User: u, WebhookBot: wb}, err
}

// UpdateWebhook Webhookを更新します
func UpdateWebhook(w Webhook, name, description *string, channelID uuid.UUID) error {
	if name != nil && w.Name() != *name {
		if len(*name) == 0 || utf8.RuneCountInString(*name) > 32 {
			return errors.New("invalid name")
		}

		if _, err := db.ID(w.BotUserID().String()).Update(&User{
			DisplayName: *name,
		}); err != nil {
			return err
		}
	}
	if description != nil && w.Description() != *description {
		if len(*description) == 0 {
			return errors.New("description is required")
		}

		if _, err := db.ID(w.ID().String()).Update(&WebhookBot{
			Description: *description,
		}); err != nil {
			return err
		}
	}
	if channelID != uuid.Nil && w.ChannelID() != channelID {
		if _, err := db.ID(w.ID().String()).Update(&WebhookBot{
			ChannelID: channelID.String(),
		}); err != nil {
			return err
		}
	}
	return nil
}

// DeleteWebhook Webhookをdbから削除
func DeleteWebhook(id uuid.UUID) (err error) {
	now := time.Now()
	_, err = db.Update(&WebhookBot{DeletedAt: &now}, &WebhookBot{ID: id.String()})
	return err
}

// GetWebhook Webhookを取得
func GetWebhook(id uuid.UUID) (Webhook, error) {
	b := &WebhookBotUser{}
	if ok, err := db.Join("INNER", "webhook_bots", "webhook_bots.bot_user_id = users.id").Where("webhook_bots.id = ? AND webhook_bots.deleted_at IS NULL", id.String()).Get(b); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return b, nil
}

// GetAllWebhooks Webhookを全て取得
func GetAllWebhooks() (arr []Webhook, err error) {
	var webhooks []*WebhookBotUser
	if err = db.Join("INNER", "webhook_bots", "webhook_bots.bot_user_id = users.id").Where("webhook_bots.deleted_at IS NULL").Find(&webhooks); err != nil {
		return nil, err
	}
	for _, v := range webhooks {
		arr = append(arr, v)
	}
	return
}

// GetWebhooksByCreator 指定した制作者のWebhookを全て取得
func GetWebhooksByCreator(id uuid.UUID) (arr []Webhook, err error) {
	var webhooks []*WebhookBotUser
	if err = db.Join("INNER", "webhook_bots", "webhook_bots.bot_user_id = users.id").Where("webhook_bots.creator_id = ? AND webhook_bots.deleted_at IS NULL", id.String()).Find(&webhooks); err != nil {
		return nil, err
	}
	for _, v := range webhooks {
		arr = append(arr, v)
	}
	return
}
