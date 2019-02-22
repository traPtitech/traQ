package impl

import (
	"encoding/base64"
	"errors"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"time"
	"unicode/utf8"
)

// WebhookBot DB用WebhookBot構造体
type WebhookBot struct {
	ID          uuid.UUID  `gorm:"type:char(36);primary_key"`
	BotUserID   uuid.UUID  `gorm:"type:char(36);unique"`
	BotUser     model.User `gorm:"foreignkey:BotUserID"`
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
func (repo *RepositoryImpl) CreateWebhook(name, description string, channelID, creatorID, iconFileID uuid.UUID) (model.Webhook, error) {
	if len(name) == 0 || utf8.RuneCountInString(name) > 32 {
		return nil, errors.New("invalid name")
	}
	if len(description) == 0 {
		return nil, errors.New("description is required")
	}
	uid := uuid.NewV4()
	bid := uuid.NewV4()

	u := &model.User{
		ID:          uid,
		Name:        "Webhook#" + base64.RawStdEncoding.EncodeToString(uid.Bytes()),
		DisplayName: name,
		Icon:        iconFileID,
		Bot:         true,
		Status:      model.UserAccountStatusActive,
		Role:        role.Bot.ID(),
	}
	wb := &WebhookBot{
		ID:          bid,
		BotUserID:   uid,
		Description: description,
		ChannelID:   channelID,
		CreatorID:   creatorID,
	}

	err := repo.transact(func(tx *gorm.DB) error {
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
	repo.hub.Publish(hub.Message{
		Name: event.UserCreated,
		Fields: hub.Fields{
			"user_id": u.ID,
			"user":    u,
		},
	})
	wb.BotUser = *u
	return wb, nil
}

// UpdateWebhook Webhookを更新します
func (repo *RepositoryImpl) UpdateWebhook(id uuid.UUID, name, description *string, channelID uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	var w WebhookBot
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&WebhookBot{ID: id}).First(&w).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return repository.ErrNotFound
			}
			return err
		}

		changes := map[string]string{}
		if description != nil {
			if len(*description) == 0 {
				return errors.New("description is required")
			}
			changes["description"] = *description
		}
		if channelID != uuid.Nil {
			changes["channel_id"] = channelID.String()
		}
		if len(changes) > 0 {
			if err := tx.Model(&WebhookBot{ID: id}).Updates(changes).Error; err != nil {
				return err
			}
		}

		if name != nil {
			if len(*name) == 0 || utf8.RuneCountInString(*name) > 32 {
				return errors.New("invalid name")
			}

			if err := tx.Model(&model.User{ID: w.BotUserID}).Update("display_name", *name).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.UserUpdated,
		Fields: hub.Fields{
			"user_id": w.BotUserID,
		},
	})
	return nil
}

// DeleteWebhook Webhookをdbから削除
func (repo *RepositoryImpl) DeleteWebhook(id uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	return repo.db.Delete(&WebhookBot{ID: id}).Error
}

// GetWebhook Webhookを取得
func (repo *RepositoryImpl) GetWebhook(id uuid.UUID) (model.Webhook, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	b := &WebhookBot{}
	if err := repo.db.Where(&WebhookBot{ID: id}).Take(b).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return b, nil
}

// GetAllWebhooks Webhookを全て取得
func (repo *RepositoryImpl) GetAllWebhooks() (arr []model.Webhook, err error) {
	var webhooks []*WebhookBot
	err = repo.db.Preload("BotUser").Find(&webhooks).Error
	if err != nil {
		return nil, err
	}
	arr = make([]model.Webhook, 0, len(webhooks))
	for _, v := range webhooks {
		arr = append(arr, v)
	}
	return arr, nil
}

// GetWebhooksByCreator 指定した制作者のWebhookを全て取得
func (repo *RepositoryImpl) GetWebhooksByCreator(creatorID uuid.UUID) (arr []model.Webhook, err error) {
	arr = make([]model.Webhook, 0)
	if creatorID == uuid.Nil {
		return arr, nil
	}

	var webhooks []*WebhookBot
	err = repo.db.Preload("BotUser").Where(&WebhookBot{CreatorID: creatorID}).Find(&webhooks).Error
	if err != nil {
		return nil, err
	}
	for _, v := range webhooks {
		arr = append(arr, v)
	}
	return arr, nil
}
