package model

import (
	"encoding/base64"
	"errors"
	"github.com/jinzhu/gorm"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/rbac/role"
	"time"
	"unicode/utf8"
)

// Webhook Webhook
type Webhook interface {
	GetID() uuid.UUID
	GetBotUserID() uuid.UUID
	GetName() string
	GetDescription() string
	GetChannelID() uuid.UUID
	GetCreatorID() uuid.UUID
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

// WebhookBot DB用WebhookBot構造体
type WebhookBot struct {
	ID          uuid.UUID  `gorm:"type:char(36);primary_key"`
	BotUserID   uuid.UUID  `gorm:"type:char(36);unique"`
	BotUser     User       `gorm:"foreignkey:BotUserID"`
	Description string     `gorm:"type:text"`
	ChannelID   uuid.UUID  `gorm:"type:char(36)"`
	CreatorID   uuid.UUID  `gorm:"type:char(36)"`
	CreatedAt   time.Time  `gorm:"precision:6"`
	UpdatedAt   time.Time  `gorm:"precision:6"`
	DeletedAt   *time.Time `gorm:"precision:6"`
}

// TableName Webhookのテーブル名
func (*WebhookBot) TableName() string {
	return "webhook_bots"
}

// GetID WebhookIDを返します
func (w *WebhookBot) GetID() uuid.UUID {
	return w.ID
}

// GetBotUserID WebhookUserのIDを返します
func (w *WebhookBot) GetBotUserID() uuid.UUID {
	return w.BotUserID
}

// GetName Webhookの名前を返します
func (w *WebhookBot) GetName() string {
	return w.BotUser.Name
}

// GetDescription Webhookの説明を返します
func (w *WebhookBot) GetDescription() string {
	return w.Description
}

// GetChannelID Webhookのデフォルト投稿チャンネルのIDを返します
func (w *WebhookBot) GetChannelID() uuid.UUID {
	return w.ChannelID
}

// GetCreatorID Webhookの製作者IDを返します
func (w *WebhookBot) GetCreatorID() uuid.UUID {
	return w.CreatorID
}

// GetCreatedAt Webhookの作成日時を返します
func (w *WebhookBot) GetCreatedAt() time.Time {
	return w.CreatedAt
}

// GetUpdatedAt Webhookの更新日時を返します
func (w *WebhookBot) GetUpdatedAt() time.Time {
	return w.UpdatedAt
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
		ID:          uid,
		Name:        "Webhook#" + base64.RawStdEncoding.EncodeToString(uid.Bytes()),
		DisplayName: name,
		Icon:        iconFileID,
		Bot:         true,
		Role:        role.Bot.ID(),
	}
	wb := &WebhookBot{
		ID:          bid,
		BotUserID:   uid,
		Description: description,
		ChannelID:   channelID,
		CreatorID:   creatorID,
	}

	err := transact(func(tx *gorm.DB) error {
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		if err := tx.Create(wb).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	wb.BotUser = *u
	return wb, nil
}

// UpdateWebhook Webhookを更新します
func UpdateWebhook(w Webhook, name, description *string, channelID uuid.UUID) error {
	if name != nil && w.GetName() != *name {
		if len(*name) == 0 || utf8.RuneCountInString(*name) > 32 {
			return errors.New("invalid name")
		}

		if err := db.Model(User{ID: w.GetBotUserID()}).Update("display_name", *name).Error; err != nil {
			return err
		}
	}
	if description != nil && w.GetDescription() != *description {
		if len(*description) == 0 {
			return errors.New("description is required")
		}

		if err := db.Model(WebhookBot{ID: w.GetID()}).Update("description", *description).Error; err != nil {
			return err
		}
	}
	if channelID != uuid.Nil && w.GetChannelID() != channelID {
		if err := db.Model(WebhookBot{ID: w.GetID()}).Update("channel_id", channelID.String()).Error; err != nil {
			return err
		}
	}
	return nil
}

// DeleteWebhook Webhookをdbから削除
func DeleteWebhook(id uuid.UUID) (err error) {
	return db.Delete(WebhookBot{ID: id}).Error
}

// GetWebhook Webhookを取得
func GetWebhook(id uuid.UUID) (Webhook, error) {
	b := &WebhookBot{}
	if err := db.Where(WebhookBot{ID: id}).Take(b).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}

// GetAllWebhooks Webhookを全て取得
func GetAllWebhooks() (arr []Webhook, err error) {
	var webhooks []*WebhookBot
	err = db.Preload("BotUser").Find(&webhooks).Error
	if err != nil {
		return nil, err
	}
	for _, v := range webhooks {
		arr = append(arr, v)
	}
	return
}

// GetWebhooksByCreator 指定した制作者のWebhookを全て取得
func GetWebhooksByCreator(creatorID uuid.UUID) (arr []Webhook, err error) {
	var webhooks []*WebhookBot
	err = db.Preload("BotUser").Where("creator_id = ?", creatorID.String()).Find(&webhooks).Error
	if err != nil {
		return nil, err
	}
	for _, v := range webhooks {
		arr = append(arr, v)
	}
	return
}
