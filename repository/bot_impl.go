package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/validator"
	"math"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

// CreateBot implements BotRepository interface.
func (repo *GormRepository) CreateBot(name, displayName, description string, creatorID uuid.UUID, webhookURL string) (*model.Bot, error) {
	if err := validator.ValidateVar(name, "required,name,max=20"); err != nil {
		return nil, ArgError("name", "invalid name")
	}
	if len(displayName) == 0 || utf8.RuneCountInString(displayName) > 32 {
		return nil, ArgError("displayName", "DisplayName must be non-empty and shorter than 33 characters")
	}
	if err := validator.ValidateVar(webhookURL, "required,url"); err != nil || !strings.HasPrefix(webhookURL, "http") {
		return nil, ArgError("webhookURL", "invalid webhookURL")
	}
	if u, _ := url.Parse(webhookURL); utils.IsPrivateHost(u.Hostname()) {
		return nil, ArgError("webhookURL", "prohibited webhook host")
	}
	if creatorID == uuid.Nil {
		return nil, ArgError("creatorID", "CreatorID is required")
	}
	if _, err := repo.GetUserByName("BOT_" + name); err == nil {
		return nil, ErrAlreadyExists
	} else if err != ErrNotFound {
		return nil, err
	}

	uid := uuid.Must(uuid.NewV4())
	bid := uuid.Must(uuid.NewV4())
	tid := uuid.Must(uuid.NewV4())
	iconID, err := repo.GenerateIconFile(name)
	if err != nil {
		return nil, err
	}

	u := &model.User{
		ID:          uid,
		Name:        "BOT_" + name,
		DisplayName: displayName,
		Icon:        iconID,
		Bot:         true,
		Status:      model.UserAccountStatusActive,
		Role:        role.Bot,
	}
	b := &model.Bot{
		ID:                bid,
		BotUserID:         uid,
		Description:       description,
		VerificationToken: utils.RandAlphabetAndNumberString(30),
		PostURL:           webhookURL,
		AccessTokenID:     tid,
		SubscribeEvents:   model.BotEvents{},
		Privileged:        false,
		State:             model.BotInactive,
		BotCode:           utils.RandAlphabetAndNumberString(30),
		CreatorID:         creatorID,
	}
	t := &model.OAuth2Token{
		ID:             tid,
		UserID:         uid,
		AccessToken:    utils.RandAlphabetAndNumberString(36),
		RefreshToken:   utils.RandAlphabetAndNumberString(36),
		RefreshEnabled: false,
		CreatedAt:      time.Now(),
		ExpiresIn:      math.MaxInt32,
		Scopes:         model.AccessScopes{"bot"},
	}

	err = repo.transact(func(tx *gorm.DB) error {
		errs := tx.Create(u).Create(t).Create(b).GetErrors()
		if len(errs) > 0 {
			return errs[0]
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
func (repo *GormRepository) UpdateBot(id uuid.UUID, args UpdateBotArgs) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	var (
		b           model.Bot
		updated     bool
		userUpdated bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.First(&b, &model.Bot{ID: id}).Error; err != nil {
			return convertError(err)
		}

		changes := map[string]interface{}{}
		if args.Description.Valid {
			changes["description"] = args.Description.String
		}
		if args.Privileged.Valid {
			changes["privileged"] = args.Privileged.Bool
		}
		if args.WebhookURL.Valid {
			w := args.WebhookURL.String
			if err := validator.ValidateVar(w, "required,url"); err != nil || !strings.HasPrefix(w, "http") {
				return ArgError("args.WebhookURL", "invalid webhookURL")
			}
			if u, _ := url.Parse(w); utils.IsPrivateHost(u.Hostname()) {
				return ArgError("args.WebhookURL", "prohibited webhook host")
			}

			changes["post_url"] = w
			changes["state"] = model.BotPaused
		}

		if len(changes) > 0 {
			if err := tx.Model(&b).Updates(changes).Error; err != nil {
				return err
			}
			updated = true
		}

		if args.DisplayName.Valid {
			if len(args.DisplayName.String) == 0 || utf8.RuneCountInString(args.DisplayName.String) > 32 {
				return ArgError("args.DisplayName", "DisplayName must be non-empty and shorter than 33 characters")
			}

			if err := tx.Model(&model.User{ID: b.BotUserID}).Update("display_name", args.DisplayName.String).Error; err != nil {
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

// SetSubscribeEventsToBot implements BotRepository interface.
func (repo *GormRepository) SetSubscribeEventsToBot(botID uuid.UUID, events model.BotEvents) error {
	if botID == uuid.Nil {
		return ErrNilID
	}
	err := repo.transact(func(tx *gorm.DB) error {
		var b model.Bot
		if err := tx.First(&b, &model.Bot{ID: botID}).Error; err != nil {
			return convertError(err)
		}
		return tx.Model(&b).Update("subscribe_events", events).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.BotSubscribeEventsChanged,
		Fields: hub.Fields{
			"bot_id": botID,
			"events": events,
		},
	})
	return nil
}

// GetAllBots implements BotRepository interface.
func (repo *GormRepository) GetAllBots() ([]*model.Bot, error) {
	bots := make([]*model.Bot, 0)
	return bots, repo.db.Find(&bots).Error
}

// GetBotByID implements BotRepository interface.
func (repo *GormRepository) GetBotByID(id uuid.UUID) (*model.Bot, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	var b model.Bot
	if err := repo.db.First(&b, &model.Bot{ID: id}).Error; err != nil {
		return nil, convertError(err)
	}
	return &b, nil
}

// GetBotByBotUserID implements BotRepository interface.
func (repo *GormRepository) GetBotByBotUserID(id uuid.UUID) (*model.Bot, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	var b model.Bot
	if err := repo.db.First(&b, &model.Bot{BotUserID: id}).Error; err != nil {
		return nil, convertError(err)
	}
	return &b, nil
}

// GetBotByCode implements BotRepository interface.
func (repo *GormRepository) GetBotByCode(code string) (*model.Bot, error) {
	if len(code) == 0 {
		return nil, ErrNotFound
	}
	var b model.Bot
	if err := repo.db.First(&b, &model.Bot{BotCode: code}).Error; err != nil {
		return nil, convertError(err)
	}
	return &b, nil
}

// GetBotsByCreator implements BotRepository interface.
func (repo *GormRepository) GetBotsByCreator(userID uuid.UUID) ([]*model.Bot, error) {
	bots := make([]*model.Bot, 0)
	if userID == uuid.Nil {
		return bots, nil
	}
	return bots, repo.db.Where(&model.Bot{CreatorID: userID}).Find(&bots).Error
}

// GetBotsByChannel implements BotRepository interface.
func (repo *GormRepository) GetBotsByChannel(channelID uuid.UUID) ([]*model.Bot, error) {
	bots := make([]*model.Bot, 0)
	if channelID == uuid.Nil {
		return bots, nil
	}
	return bots, repo.db.
		Where("id IN ?", repo.db.
			Model(&model.BotJoinChannel{}).
			Select("bot_id").
			Where(&model.BotJoinChannel{ChannelID: channelID}).
			SubQuery()).
		Find(&bots).
		Error
}

// ChangeBotState implements BotRepository interface.
func (repo *GormRepository) ChangeBotState(id uuid.UUID, state model.BotState) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	var changed bool
	err := repo.transact(func(tx *gorm.DB) error {
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
func (repo *GormRepository) ReissueBotTokens(id uuid.UUID) (*model.Bot, error) {
	if id == uuid.Nil {
		return nil, ErrNilID
	}
	var bot *model.Bot
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.First(&bot, &model.Bot{ID: id}).Error; err != nil {
			return convertError(err)
		}

		bot.State = model.BotPaused
		bot.BotCode = utils.RandAlphabetAndNumberString(30)
		bot.VerificationToken = utils.RandAlphabetAndNumberString(30)

		if err := tx.Delete(&model.OAuth2Token{ID: bot.AccessTokenID}).Error; err != nil {
			return err
		}

		tid := uuid.Must(uuid.NewV4())
		t := &model.OAuth2Token{
			ID:             tid,
			UserID:         bot.BotUserID,
			AccessToken:    utils.RandAlphabetAndNumberString(36),
			RefreshToken:   utils.RandAlphabetAndNumberString(36),
			RefreshEnabled: false,
			CreatedAt:      time.Now(),
			ExpiresIn:      math.MaxInt32,
			Scopes:         model.AccessScopes{"bot"},
		}
		bot.AccessTokenID = tid

		errs := tx.Create(t).Save(bot).GetErrors()
		if len(errs) > 0 {
			return errs[0]
		}
		return nil
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
	return bot, nil
}

// DeleteBot implements BotRepository interface.
func (repo *GormRepository) DeleteBot(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	err := repo.transact(func(tx *gorm.DB) error {
		var b model.Bot
		if err := tx.First(&b, &model.Bot{ID: id}).Error; err != nil {
			return convertError(err)
		}

		errs := tx.Model(&model.User{ID: b.BotUserID}).Update("status", model.UserAccountStatusDeactivated).New().
			Delete(&model.BotJoinChannel{BotID: id}).
			Delete(&model.OAuth2Token{ID: b.AccessTokenID}).
			Delete(&model.Bot{ID: id}).
			GetErrors()
		if len(errs) > 0 {
			return errs[0]
		}
		return nil
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
func (repo *GormRepository) AddBotToChannel(botID, channelID uuid.UUID) error {
	if botID == uuid.Nil || channelID == uuid.Nil {
		return ErrNilID
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
func (repo *GormRepository) RemoveBotFromChannel(botID, channelID uuid.UUID) error {
	if botID == uuid.Nil || channelID == uuid.Nil {
		return ErrNilID
	}
	result := repo.db.Delete(&model.BotJoinChannel{BotID: botID, ChannelID: channelID})
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
func (repo *GormRepository) GetParticipatingChannelIDsByBot(botID uuid.UUID) ([]uuid.UUID, error) {
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
func (repo *GormRepository) WriteBotEventLog(log *model.BotEventLog) error {
	if log == nil || log.RequestID == uuid.Nil {
		return nil
	}
	return repo.db.Create(log).Error
}

// GetBotEventLogs implements BotRepository interface.
func (repo *GormRepository) GetBotEventLogs(botID uuid.UUID, limit, offset int) ([]*model.BotEventLog, error) {
	logs := make([]*model.BotEventLog, 0)
	if botID == uuid.Nil {
		return logs, nil
	}
	return logs, repo.db.Where(&model.BotEventLog{BotID: botID}).
		Order("date_time DESC").
		Scopes(limitAndOffset(limit, offset)).
		Find(&logs).
		Error
}
