package repository

import (
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/validator"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

// CreateBot implements BotRepository interface.
func (repo *GormRepository) CreateBot(name, displayName, description string, creatorID uuid.UUID, webhookURL string) (*model.Bot, error) {
	if err := vd.Validate(name, validator.BotUserNameRuleRequired...); err != nil {
		return nil, ArgError("name", "invalid name")
	}
	if len(displayName) == 0 || utf8.RuneCountInString(displayName) > 32 {
		return nil, ArgError("displayName", "DisplayName must be non-empty and shorter than 33 characters")
	}
	if err := vd.Validate(webhookURL, vd.Required, is.URL, validator.NotInternalURL); err != nil || !strings.HasPrefix(webhookURL, "http") {
		return nil, ArgError("webhookURL", "invalid webhookURL")
	}
	if creatorID == uuid.Nil {
		return nil, ArgError("creatorID", "CreatorID is required")
	}
	if _, err := repo.GetUserByName("BOT_"+name, false); err == nil {
		return nil, ErrAlreadyExists
	} else if err != ErrNotFound {
		return nil, err
	}

	uid := uuid.Must(uuid.NewV4())
	bid := uuid.Must(uuid.NewV4())
	tid := uuid.Must(uuid.NewV4())
	iconID, err := GenerateIconFile(repo, name)
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
		Profile:     &model.UserProfile{UserID: uid},
	}
	b := &model.Bot{
		ID:                bid,
		BotUserID:         uid,
		Description:       description,
		VerificationToken: random.SecureAlphaNumeric(30),
		PostURL:           webhookURL,
		AccessTokenID:     tid,
		SubscribeEvents:   model.BotEvents{},
		Privileged:        false,
		State:             model.BotInactive,
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

	err = repo.db.Transaction(func(tx *gorm.DB) error {
		errs := tx.Create(u).Create(u.Profile).Create(t).Create(b).GetErrors()
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
	err := repo.db.Transaction(func(tx *gorm.DB) error {
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
			if err := vd.Validate(w, vd.Required, is.URL, validator.NotInternalURL); err != nil || !strings.HasPrefix(w, "http") {
				return ArgError("args.WebhookURL", "invalid webhookURL")
			}
			changes["post_url"] = w
			changes["state"] = model.BotPaused
		}
		if args.CreatorID.Valid {
			// 作成者検証
			user, err := getUser(tx, false, "id = ?", args.CreatorID.UUID)
			if err != nil {
				if err == ErrNotFound {
					return ArgError("args.CreatorID", "the Creator is not found")
				}
				return err
			}
			if !user.IsActive() || user.IsBot() {
				return ArgError("args.CreatorID", "invalid User")
			}

			changes["creator_id"] = args.CreatorID.UUID
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

// GetBots implements BotRepository interface.
func (repo *GormRepository) GetBots(query BotsQuery) ([]*model.Bot, error) {
	bots := make([]*model.Bot, 0)
	tx := repo.db.Table("bots")

	if query.IsPrivileged.Valid {
		tx = tx.Where("bots.privileged = ?", query.IsPrivileged.Bool)
	}
	if query.IsActive.Valid {
		if query.IsActive.Bool {
			tx = tx.Where("bots.state = ?", model.BotActive)
		} else {
			tx = tx.Where("bots.state != ?", model.BotActive)
		}
	}
	if query.Creator.Valid {
		tx = tx.Where("bots.creator_id = ?", query.Creator.UUID)
	}
	if query.IsCMemberOf.Valid {
		tx = tx.Joins("INNER JOIN bot_join_channels ON bot_join_channels.bot_id = bots.id AND bot_join_channels.channel_id = ?", query.IsCMemberOf.UUID)
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
func (repo *GormRepository) GetBotByID(id uuid.UUID) (*model.Bot, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	return getBot(repo.db, &model.Bot{ID: id})
}

// GetBotByBotUserID implements BotRepository interface.
func (repo *GormRepository) GetBotByBotUserID(id uuid.UUID) (*model.Bot, error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	return getBot(repo.db, &model.Bot{BotUserID: id})
}

// GetBotByCode implements BotRepository interface.
func (repo *GormRepository) GetBotByCode(code string) (*model.Bot, error) {
	if len(code) == 0 {
		return nil, ErrNotFound
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
func (repo *GormRepository) ChangeBotState(id uuid.UUID, state model.BotState) error {
	if id == uuid.Nil {
		return ErrNilID
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
func (repo *GormRepository) ReissueBotTokens(id uuid.UUID) (*model.Bot, error) {
	if id == uuid.Nil {
		return nil, ErrNilID
	}
	var bot model.Bot
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&bot, &model.Bot{ID: id}).Error; err != nil {
			return convertError(err)
		}

		bot.State = model.BotPaused
		bot.BotCode = random.AlphaNumeric(30)
		bot.VerificationToken = random.SecureAlphaNumeric(30)

		if err := tx.Delete(&model.OAuth2Token{ID: bot.AccessTokenID}).Error; err != nil {
			return err
		}

		tid := uuid.Must(uuid.NewV4())
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

		errs := tx.Create(t).Save(&bot).GetErrors()
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
	return &bot, nil
}

// DeleteBot implements BotRepository interface.
func (repo *GormRepository) DeleteBot(id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
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
