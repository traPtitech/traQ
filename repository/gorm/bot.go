package gorm

import (
	"math"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/utils/gormutil"
	"github.com/traPtitech/traQ/utils/random"
)

// CreateBot implements BotRepository interface.
func (repo *Repository) CreateBot(name, displayName, description string, iconFileID, creatorID uuid.UUID, mode model.BotMode, state model.BotState, webhookURL string) (*model.Bot, error) {
	uid := uuid.Must(uuid.NewV7())
	bid := uuid.Must(uuid.NewV7())
	tid := uuid.Must(uuid.NewV7())
	u := &model.User{
		ID:          uid,
		Name:        "BOT_" + name,
		DisplayName: displayName,
		Icon:        iconFileID,
		Bot:         true,
		Status:      model.UserAccountStatusActive,
		Role:        role.Bot,
		Profile:     &model.UserProfile{UserID: uid},
	}
	b := &model.Bot{
		ID:                bid,
		BotUserID:         uid,
		Description:       description,
		VerificationToken: random.SecureAlphaNumeric(30),
		PostURL:           webhookURL,
		AccessTokenID:     tid,
		SubscribeEvents:   model.BotEventTypes{},
		Privileged:        false,
		Mode:              mode,
		State:             state,
		BotCode:           random.AlphaNumeric(30),
		CreatorID:         creatorID,
	}
	scopes := model.AccessScopes{}
	scopes.Add("bot")
	t := &model.OAuth2Token{
		ID:             tid,
		UserID:         uid,
		AccessToken:    random.SecureAlphaNumeric(36),
		RefreshToken:   random.SecureAlphaNumeric(36),
		RefreshEnabled: false,
		CreatedAt:      time.Now(),
		ExpiresIn:      math.MaxInt32,
		Scopes:         scopes,
	}

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		if err := tx.Create(t).Error; err != nil {
			return err
		}
		return tx.Create(b).Error
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
	repo.hub.Publish(hub.Message{
		Name: event.BotCreated,
		Fields: hub.Fields{
			"bot_id": b.ID,
			"bot":    b,
		},
	})
	return b, nil
}

// UpdateBot implements BotRepository interface.
func (repo *Repository) UpdateBot(id uuid.UUID, args repository.UpdateBotArgs) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	var (
		u           model.User
		b           model.Bot
		updated     bool
		userUpdated bool
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&b, &model.Bot{ID: id}).Error; err != nil {
			return convertError(err)
		}

		changes := map[string]interface{}{}
		if args.Description.Valid {
			changes["description"] = args.Description.V
		}
		if args.Privileged.Valid {
			changes["privileged"] = args.Privileged.V
		}
		if args.Mode.Valid {
			changes["mode"] = args.Mode.V
		}
		if args.WebhookURL.Valid {
			w := args.WebhookURL.V
			changes["post_url"] = w
			changes["state"] = model.BotPaused
		}
		if args.CreatorID.Valid {
			changes["creator_id"] = args.CreatorID.V
		}
		if args.SubscribeEvents != nil {
			changes["subscribe_events"] = args.SubscribeEvents
		}

		if len(changes) > 0 {
			if err := tx.Model(&b).Updates(changes).Error; err != nil {
				return err
			}
			updated = true
		}

		if args.DisplayName.Valid {
			if err := tx.Model(&model.User{ID: b.BotUserID}).Update("display_name", args.DisplayName.V).Error; err != nil {
				return err
			}
			userUpdated = true
		}

		if args.Bio.Valid {
			if err := tx.Preload("Profile").First(&u, model.User{ID: b.BotUserID}).Error; err != nil {
				return convertError(err)
			}

			if err := tx.Model(u.Profile).Update("bio", args.Bio.V).Error; err != nil {
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
				"user_id": b.BotUserID,
			},
		})
	}
	if updated || userUpdated {
		repo.hub.Publish(hub.Message{
			Name: event.BotUpdated,
			Fields: hub.Fields{
				"bot_id": b.ID,
			},
		})
	}
	return nil
}

// GetBots implements BotRepository interface.
func (repo *Repository) GetBots(query repository.BotsQuery) ([]*model.Bot, error) {
	bots := make([]*model.Bot, 0)
	tx := repo.db.Table("bots")

	if query.IsPrivileged.Valid {
		tx = tx.Where("bots.privileged = ?", query.IsPrivileged.V)
	}
	if query.IsActive.Valid {
		if query.IsActive.V {
			tx = tx.Where("bots.state = ?", model.BotActive)
		} else {
			tx = tx.Where("bots.state != ?", model.BotActive)
		}
	}
	if query.Creator.Valid {
		tx = tx.Where("bots.creator_id = ?", query.Creator.V)
	}
	if query.ID.Valid {
		tx = tx.Where("bots.id = ?", query.ID.V)
	}
	if query.UserID.Valid {
		tx = tx.Where("bots.bot_user_id = ?", query.UserID.V)
	}
	if query.IsCMemberOf.Valid {
		tx = tx.Joins("INNER JOIN bot_join_channels ON bot_join_channels.bot_id = bots.id AND bot_join_channels.channel_id = ?", query.IsCMemberOf.V)
	}
	if len(query.SubscribeEvents) == 0 {
		return bots, tx.Find(&bots).Error
	}

	// MEMO SubscribeEventsを正規化したほうがいいかもしれない
	if err := tx.Find(&bots).Error; err != nil {
		return nil, err
	}
	result := make([]*model.Bot, 0, len(bots))
BotsFor:
	for _, v := range bots {
		for e := range query.SubscribeEvents {
			if !v.SubscribeEvents.Contains(e) {
				continue BotsFor
			}
		}
		result = append(result, v)
	}
	return result, nil
}

// GetBotByID implements BotRepository interface.
func (repo *Repository) GetBotByID(id uuid.UUID) (*model.Bot, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	return getBot(repo.db, &model.Bot{ID: id})
}

// GetBotByBotUserID implements BotRepository interface.
func (repo *Repository) GetBotByBotUserID(id uuid.UUID) (*model.Bot, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	return getBot(repo.db, &model.Bot{BotUserID: id})
}

// GetBotByCode implements BotRepository interface.
func (repo *Repository) GetBotByCode(code string) (*model.Bot, error) {
	if len(code) == 0 {
		return nil, repository.ErrNotFound
	}
	return getBot(repo.db, &model.Bot{BotCode: code})
}

func getBot(tx *gorm.DB, where interface{}) (*model.Bot, error) {
	var b model.Bot
	if err := tx.First(&b, where).Error; err != nil {
		return nil, convertError(err)
	}
	return &b, nil
}

// ChangeBotState implements BotRepository interface.
func (repo *Repository) ChangeBotState(id uuid.UUID, state model.BotState) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	var changed bool

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var b model.Bot
		if err := tx.Take(&b, &model.Bot{ID: id}).Error; err != nil {
			return convertError(err)
		}
		if b.State == state {
			return nil
		}
		changed = true
		return tx.Model(&b).Update("state", state).Error
	})
	if err != nil {
		return err
	}
	if changed {
		repo.hub.Publish(hub.Message{
			Name: event.BotStateChanged,
			Fields: hub.Fields{
				"bot_id": id,
				"state":  state,
			},
		})
	}
	return nil
}

// ReissueBotTokens implements BotRepository interface.
func (repo *Repository) ReissueBotTokens(id uuid.UUID) (*model.Bot, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNilID
	}
	var bot model.Bot
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&bot, &model.Bot{ID: id}).Error; err != nil {
			return convertError(err)
		}

		if bot.Mode == model.BotModeHTTP {
			bot.State = model.BotPaused
		}
		bot.BotCode = random.AlphaNumeric(30)
		bot.VerificationToken = random.SecureAlphaNumeric(30)

		if err := tx.Delete(&model.OAuth2Token{ID: bot.AccessTokenID}).Error; err != nil {
			return err
		}

		tid := uuid.Must(uuid.NewV7())
		scopes := model.AccessScopes{}
		scopes.Add("bot")
		t := &model.OAuth2Token{
			ID:             tid,
			UserID:         bot.BotUserID,
			AccessToken:    random.SecureAlphaNumeric(36),
			RefreshToken:   random.SecureAlphaNumeric(36),
			RefreshEnabled: false,
			CreatedAt:      time.Now(),
			ExpiresIn:      math.MaxInt32,
			Scopes:         scopes,
		}
		bot.AccessTokenID = tid

		if err := tx.Create(t).Error; err != nil {
			return err
		}
		return tx.Save(&bot).Error
	})
	if err != nil {
		return nil, err
	}
	repo.hub.Publish(hub.Message{
		Name: event.BotStateChanged,
		Fields: hub.Fields{
			"bot_id": id,
			"state":  bot.State,
		},
	})
	return &bot, nil
}

// DeleteBot implements BotRepository interface.
func (repo *Repository) DeleteBot(id uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var b model.Bot
		if err := tx.First(&b, &model.Bot{ID: id}).Error; err != nil {
			return convertError(err)
		}

		if err := tx.
			Model(&model.User{ID: b.BotUserID}).
			Update("status", model.UserAccountStatusDeactivated).
			Error; err != nil {
			return err
		}

		if err := tx.Delete(&model.BotJoinChannel{}, &model.BotJoinChannel{BotID: id}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.OAuth2Token{}, &model.OAuth2Token{ID: b.AccessTokenID}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Bot{}, &model.Bot{ID: id}).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.BotDeleted,
		Fields: hub.Fields{
			"bot_id": id,
		},
	})
	return nil
}

// AddBotToChannel implements BotRepository interface.
func (repo *Repository) AddBotToChannel(botID, channelID uuid.UUID) error {
	if botID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	var b model.BotJoinChannel
	result := repo.db.FirstOrCreate(&b, &model.BotJoinChannel{BotID: botID, ChannelID: channelID})
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.BotJoined,
			Fields: hub.Fields{
				"bot_id":     botID,
				"channel_id": channelID,
			},
		})
	}
	return result.Error
}

// RemoveBotFromChannel implements BotRepository interface.
func (repo *Repository) RemoveBotFromChannel(botID, channelID uuid.UUID) error {
	if botID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	result := repo.db.Delete(&model.BotJoinChannel{}, &model.BotJoinChannel{BotID: botID, ChannelID: channelID})
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.BotLeft,
			Fields: hub.Fields{
				"bot_id":     botID,
				"channel_id": channelID,
			},
		})
	}
	return result.Error
}

// GetParticipatingChannelIDsByBot implements BotRepository interface.
func (repo *Repository) GetParticipatingChannelIDsByBot(botID uuid.UUID) ([]uuid.UUID, error) {
	channels := make([]uuid.UUID, 0)
	if botID == uuid.Nil {
		return channels, nil
	}
	return channels, repo.db.
		Model(&model.BotJoinChannel{}).
		Where(&model.BotJoinChannel{BotID: botID}).
		Pluck("channel_id", &channels).
		Error
}

// WriteBotEventLog implements BotRepository interface.
func (repo *Repository) WriteBotEventLog(log *model.BotEventLog) error {
	if log == nil || log.RequestID == uuid.Nil {
		return nil
	}
	return repo.db.Create(log).Error
}

// GetBotEventLogs implements BotRepository interface.
func (repo *Repository) GetBotEventLogs(botID uuid.UUID, limit, offset int) ([]*model.BotEventLog, error) {
	logs := make([]*model.BotEventLog, 0)
	if botID == uuid.Nil {
		return logs, nil
	}
	return logs, repo.db.Where(&model.BotEventLog{BotID: botID}).
		Order("date_time DESC").
		Scopes(gormutil.LimitAndOffset(limit, offset)).
		Find(&logs).
		Error
}

// PurgeBotEventLogs implements BotRepository interface.
func (repo *Repository) PurgeBotEventLogs(before time.Time) error {
	return repo.db.Delete(&model.BotEventLog{}, "date_time < ?", before).Error
}
