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
	"strings"
	"time"
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
		Status:            model.BotInactive,
		BotCode:           utils.RandAlphabetAndNumberString(30),
		CreatorID:         creatorID,
	}
	t := &model.OAuth2Token{
		ID:          tid,
		UserID:      uid,
		AccessToken: utils.RandAlphabetAndNumberString(36),
		CreatedAt:   time.Now(),
		ExpiresIn:   math.MaxInt32,
		Scopes:      model.AccessScopes{"bot"},
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

// ChangeBotStatus Botの状態を変更します
func (repo *RepositoryImpl) ChangeBotStatus(id uuid.UUID, status model.BotStatus) error {
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
		if b.Status == status {
			return nil
		}
		changed = true
		return tx.Model(&b).Update("status", status).Error
	})
	if err != nil {
		return err
	}
	if changed {
		repo.hub.Publish(hub.Message{
			Name: event.BotStatusChanged,
			Fields: hub.Fields{
				"bot_id": id,
				"status": status,
			},
		})
	}
	return nil
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
