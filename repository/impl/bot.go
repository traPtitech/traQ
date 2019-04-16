package impl

import (
	"errors"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/validator"
	"math"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

// CreateBot Botを作成します
func (repo *RepositoryImpl) CreateBot(name, displayName, description string, creatorID uuid.UUID, webhookURL string) (*model.Bot, error) {
	if err := validator.ValidateVar(name, "required,name,max=20"); err != nil {
		return nil, errors.New("invalid name")
	}
	if err := validator.ValidateVar(displayName, "required,max=64"); err != nil {
		return nil, errors.New("invalid displayName")
	}
	if err := validator.ValidateVar(webhookURL, "required,url"); err != nil || !strings.HasPrefix(webhookURL, "http") {
		return nil, errors.New("invalid webhookURL")
	}
	if u, _ := url.Parse(webhookURL); utils.IsPrivateHost(u.Hostname()) {
		return nil, errors.New("prohibited webhook host")
	}
	if creatorID == uuid.Nil {
		return nil, errors.New("creatorID is nil")
	}
	if _, err := repo.GetUserByName("BOT_" + name); err == nil {
		return nil, repository.ErrAlreadyExists
	} else if err != repository.ErrNotFound {
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
		Role:        role.Bot.ID(),
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

// UpdateBot Botを更新します
func (repo *RepositoryImpl) UpdateBot(id uuid.UUID, args repository.UpdateBotArgs) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	var (
		b           model.Bot
		updated     bool
		userUpdated bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&model.Bot{ID: id}).First(&b).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return repository.ErrNotFound
			}
			return err
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
				return errors.New("invalid webhookURL")
			}
			if u, _ := url.Parse(w); utils.IsPrivateHost(u.Hostname()) {
				return errors.New("prohibited webhook host")
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
				return errors.New("invalid name")
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

// SetSubscribeEventsToBot Botの購読イベントを変更します
func (repo *RepositoryImpl) SetSubscribeEventsToBot(botID uuid.UUID, events model.BotEvents) error {
	if botID == uuid.Nil {
		return repository.ErrNilID
	}
	err := repo.transact(func(tx *gorm.DB) error {
		var b model.Bot
		if err := tx.Take(&b, &model.Bot{ID: botID}).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return repository.ErrNotFound
			}
			return err
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

// GetAllBots 全てのBotを取得します
func (repo *RepositoryImpl) GetAllBots() ([]*model.Bot, error) {
	bots := make([]*model.Bot, 0)
	return bots, repo.db.Find(&bots).Error
}

// GetBotByID Botを取得します
func (repo *RepositoryImpl) GetBotByID(id uuid.UUID) (*model.Bot, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	var b model.Bot
	if err := repo.db.Where(&model.Bot{ID: id}).First(&b).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}

// GetBotByCode BotCodeからBotを取得します
func (repo *RepositoryImpl) GetBotByCode(code string) (*model.Bot, error) {
	if len(code) == 0 {
		return nil, repository.ErrNotFound
	}
	var b model.Bot
	if err := repo.db.Where(&model.Bot{BotCode: code}).First(&b).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return &b, nil
}

// GetBotsByCreator 指定したCreatorのBotを取得します
func (repo *RepositoryImpl) GetBotsByCreator(userID uuid.UUID) ([]*model.Bot, error) {
	bots := make([]*model.Bot, 0)
	if userID == uuid.Nil {
		return bots, nil
	}
	return bots, repo.db.Where(&model.Bot{CreatorID: userID}).Find(&bots).Error
}

// GetBotsByChannel 指定したチャンネルに参加しているBotを取得します
func (repo *RepositoryImpl) GetBotsByChannel(channelID uuid.UUID) ([]*model.Bot, error) {
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

// ChangeBotState Botの状態を変更します
func (repo *RepositoryImpl) ChangeBotState(id uuid.UUID, state model.BotState) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	var changed bool
	err := repo.transact(func(tx *gorm.DB) error {
		var b model.Bot
		if err := tx.Take(&b, &model.Bot{ID: id}).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return repository.ErrNotFound
			}
			return err
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

// ReissueBotTokens Botの各種トークンを再発行します
func (repo *RepositoryImpl) ReissueBotTokens(id uuid.UUID) (*model.Bot, error) {
	if id == uuid.Nil {
		return nil, repository.ErrNilID
	}
	var bot *model.Bot
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Take(&bot, &model.Bot{ID: id}).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return repository.ErrNotFound
			}
			return err
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

// DeleteBot Botを削除します
func (repo *RepositoryImpl) DeleteBot(id uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	err := repo.transact(func(tx *gorm.DB) error {
		var b model.Bot
		if err := tx.Take(&b, &model.Bot{ID: id}).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return repository.ErrNotFound
			}
			return err
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

// AddBotToChannel Botをチャンネルに参加させます
func (repo *RepositoryImpl) AddBotToChannel(botID, channelID uuid.UUID) error {
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

// RemoveBotFromChannel Botをチャンネルから退出させます
func (repo *RepositoryImpl) RemoveBotFromChannel(botID, channelID uuid.UUID) error {
	if botID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
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

// GetParticipatingChannelIDsByBot Botが参加しているチャンネルのIDを取得します
func (repo *RepositoryImpl) GetParticipatingChannelIDsByBot(botID uuid.UUID) ([]uuid.UUID, error) {
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

// WriteBotEventLog Botイベントログを書き込みます
func (repo *RepositoryImpl) WriteBotEventLog(log *model.BotEventLog) error {
	if log == nil || log.RequestID == uuid.Nil {
		return nil
	}
	return repo.db.Create(log).Error
}

// GetBotEventLogs Botイベントログを取得します
func (repo *RepositoryImpl) GetBotEventLogs(botID uuid.UUID, limit, offset int) ([]*model.BotEventLog, error) {
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
