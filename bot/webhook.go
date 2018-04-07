package bot

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// Webhook Webhook構造体
type Webhook struct {
	ID          uuid.UUID
	BotUserID   uuid.UUID
	Name        string `validate:"max=32,required"`
	Description string `validate:"required"`
	ChannelID   uuid.UUID
	IconFileID  uuid.UUID
	CreatorID   uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	IsValid     bool
}

// Validate 構造体を検証します
func (w *Webhook) Validate() error {
	return validator.ValidateStruct(w)
}

// CreateWebhook Webhookを作成します
func (h *Dao) CreateWebhook(name, description string, channelID, creatorID, iconFileID uuid.UUID) (Webhook, error) {
	w := &Webhook{
		ID:          uuid.NewV4(),
		BotUserID:   uuid.NewV4(),
		Name:        name,
		Description: description,
		ChannelID:   channelID,
		IconFileID:  iconFileID,
		CreatorID:   creatorID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		IsValid:     true,
	}

	if err := w.Validate(); err != nil {
		return Webhook{}, err
	}
	if err := h.store.SaveWebhook(w); err != nil {
		return Webhook{}, err
	}

	return *w, nil
}

// GetWebhook Webhookを取得します
func (h *Dao) GetWebhook(id uuid.UUID) (Webhook, bool) {
	return h.store.GetWebhook(id)
}

// GetAllWebhooks 全てのWebhookを取得します
func (h *Dao) GetAllWebhooks() []Webhook {
	return h.store.GetAllWebhooks()
}

// GetWebhooksByCreator 指定した登録者のWebhookを全て取得します
func (h *Dao) GetWebhooksByCreator(userID uuid.UUID) (result []Webhook) {
	for _, v := range h.GetAllWebhooks() {
		if v.CreatorID == userID {
			result = append(result, v)
		}
	}
	return result
}

// UpdateWebhook Webhookを更新します。ID, BotUserID, CreatorID, CreatedAtは更新されません。
func (h *Dao) UpdateWebhook(webhook *Webhook) (err error) {
	err = webhook.Validate()
	if err != nil {
		return err
	}
	return h.store.SaveWebhook(webhook)
}
