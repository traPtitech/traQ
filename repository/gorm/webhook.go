package gorm

import (
	"encoding/base64"
	"unicode/utf8"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/rbac/role"
)

// CreateWebhook implements WebhookRepository interface.
func (repo *Repository) CreateWebhook(name, description string, channelID, iconFileID, creatorID uuid.UUID, secret string) (model.Webhook, error) {
	if len(name) == 0 || utf8.RuneCountInString(name) > 32 {
		return nil, repository.ArgError("name", "Name must be non-empty and shorter than 33 characters")
	}

	uid := uuid.Must(uuid.NewV7())
	bid := uuid.Must(uuid.NewV7())
	u := &model.User{
		ID:          uid,
		Name:        "Webhook#" + base64.RawURLEncoding.EncodeToString(uid.Bytes()),
		DisplayName: name,
		Icon:        iconFileID,
		Bot:         true,
		Status:      model.UserAccountStatusActive,
		Role:        role.Bot,
		Profile:     &model.UserProfile{UserID: uid},
	}
	wb := &model.WebhookBot{
		ID:          bid,
		BotUserID:   uid,
		Description: description,
		Secret:      secret,
		ChannelID:   channelID,
		CreatorID:   creatorID,
	}

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		// チャンネル検証
		var ch model.Channel
		if err := tx.First(&ch, &model.Channel{ID: channelID}).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return repository.ArgError("channelID", "the Channel is not found")
			}
			return err
		}
		if !ch.IsPublic {
			return repository.ArgError("channelID", "private channels are not allowed")
		}

		// Create user, user_profile
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		// Create webhook_bot
		return tx.Create(wb).Error
	})
	if err != nil {
		return nil, err
	}
	wb.BotUser = *u
	repo.hub.Publish(hub.Message{
		Name: event.UserCreated,
		Fields: hub.Fields{
			"user_id": u.ID,
			"user":    u,
		},
	})
	repo.hub.Publish(hub.Message{
		Name: event.WebhookCreated,
		Fields: hub.Fields{
			"webhook_id": wb.ID,
			"webhook":    wb,
		},
	})
	return wb, nil
}

// UpdateWebhook implements WebhookRepository interface.
func (repo *Repository) UpdateWebhook(id uuid.UUID, args repository.UpdateWebhookArgs) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	var (
		w           model.WebhookBot
		updated     bool
		userUpdated bool
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(&model.WebhookBot{ID: id}).First(&w).Error; err != nil {
			return convertError(err)
		}

		changes := map[string]interface{}{}
		if args.Description.Valid {
			changes["description"] = args.Description.V
		}
		if args.ChannelID.Valid {
			// チャンネル検証
			var ch model.Channel
			if err := tx.First(&ch, &model.Channel{ID: args.ChannelID.V}).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return repository.ArgError("args.ChannelID", "the Channel is not found")
				}
				return err
			}
			if !ch.IsPublic {
				return repository.ArgError("args.ChannelID", "private channels are not allowed")
			}

			changes["channel_id"] = args.ChannelID.V
		}
		if args.Secret.Valid {
			changes["secret"] = args.Secret.V
		}
		if args.CreatorID.Valid {
			// 作成者検証
			user, err := repo.GetUser(args.CreatorID.V, false)
			if err != nil {
				if err == repository.ErrNotFound {
					return repository.ArgError("args.CreatorID", "the Creator is not found")
				}
				return err
			}
			if !user.IsActive() || user.IsBot() {
				return repository.ArgError("args.CreatorID", "invalid User")
			}

			changes["creator_id"] = args.CreatorID.V
		}
		if len(changes) > 0 {
			if err := tx.Model(&model.WebhookBot{ID: id}).Updates(changes).Error; err != nil {
				return err
			}
			updated = true
		}

		if args.Name.Valid {
			if len(args.Name.V) == 0 || utf8.RuneCountInString(args.Name.V) > 32 {
				return repository.ArgError("args.Name", "Name must be non-empty and shorter than 33 characters")
			}

			if err := tx.Model(&model.User{ID: w.BotUserID}).Update("display_name", args.Name.V).Error; err != nil {
				return err
			}
			userUpdated = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if userUpdated {
		repo.hub.Publish(hub.Message{
			Name: event.UserUpdated,
			Fields: hub.Fields{
				"user_id": w.BotUserID,
			},
		})
	}
	if updated || userUpdated {
		repo.hub.Publish(hub.Message{
			Name: event.WebhookUpdated,
			Fields: hub.Fields{
				"webhook_id": w.ID,
			},
		})
	}
	return nil
}

// DeleteWebhook implements WebhookRepository interface.
func (repo *Repository) DeleteWebhook(id uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var b model.WebhookBot
		if err := tx.Take(&b, &model.WebhookBot{ID: id}).Error; err != nil {
			return convertError(err)
		}

		if err := tx.Delete(&model.WebhookBot{ID: id}).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where(&model.User{ID: b.BotUserID}).Update("status", model.UserAccountStatusDeactivated).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.WebhookDeleted,
		Fields: hub.Fields{
			"webhook_id": id,
		},
	})
	return nil
}

// GetWebhook implements WebhookRepository interface.
func (repo *Repository) GetWebhook(id uuid.UUID) (model.Webhook, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	b := &model.WebhookBot{}
	if err := repo.db.Preload("BotUser").Where(&model.WebhookBot{ID: id}).Take(b).Error; err != nil {
		return nil, convertError(err)
	}
	return b, nil
}

// GetWebhookByBotUserID implements WebhookRepository interface.
func (repo *Repository) GetWebhookByBotUserID(id uuid.UUID) (model.Webhook, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	b := &model.WebhookBot{}
	if err := repo.db.Preload("BotUser").Where(&model.WebhookBot{BotUserID: id}).Take(b).Error; err != nil {
		return nil, convertError(err)
	}
	return b, nil
}

// GetAllWebhooks implements WebhookRepository interface.
func (repo *Repository) GetAllWebhooks() (arr []model.Webhook, err error) {
	var webhooks []*model.WebhookBot
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

// GetWebhooksByCreator implements WebhookRepository interface.
func (repo *Repository) GetWebhooksByCreator(creatorID uuid.UUID) (arr []model.Webhook, err error) {
	arr = make([]model.Webhook, 0)
	if creatorID == uuid.Nil {
		return arr, nil
	}

	var webhooks []*model.WebhookBot
	err = repo.db.Preload("BotUser").Where(&model.WebhookBot{CreatorID: creatorID}).Find(&webhooks).Error
	if err != nil {
		return nil, err
	}
	for _, v := range webhooks {
		arr = append(arr, v)
	}
	return arr, nil
}
