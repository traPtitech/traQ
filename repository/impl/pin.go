package impl

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

// CreatePin ピン留めを作成する
func (repo *RepositoryImpl) CreatePin(messageID, userID uuid.UUID) (uuid.UUID, error) {
	if messageID == uuid.Nil || userID == uuid.Nil {
		return uuid.Nil, repository.ErrNilID
	}
	var p model.Pin
	err := repo.db.
		Where(&model.Pin{MessageID: messageID}).
		Attrs(&model.Pin{ID: uuid.NewV4(), UserID: userID}).
		FirstOrCreate(&p).
		Error
	if err != nil {
		return uuid.Nil, err
	}
	repo.hub.Publish(hub.Message{
		Name: event.MessagePinned,
		Fields: hub.Fields{
			"message_id": messageID,
			"pin_id":     p.ID,
		},
	})
	return p.ID, err
}

// GetPin ピン留めを取得する
func (repo *RepositoryImpl) GetPin(id uuid.UUID) (p *model.Pin, err error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	p = &model.Pin{}
	err = repo.db.Preload("Message").Where(&model.Pin{ID: id}).Take(p).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return p, err
}

// IsPinned 指定したメッセージがピン留めされているかを取得する
func (repo *RepositoryImpl) IsPinned(messageID uuid.UUID) (bool, error) {
	if messageID == uuid.Nil {
		return false, nil
	}
	c := 0
	err := repo.db.
		Model(&model.Pin{}).
		Where(&model.Pin{MessageID: messageID}).
		Limit(1).
		Count(&c).
		Error
	return c > 0, err
}

// DeletePin ピン留めを削除する
func (repo *RepositoryImpl) DeletePin(id uuid.UUID) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	var (
		pin model.Pin
		ok  bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&model.Pin{ID: id}).First(&pin).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return nil
			}
			return err
		}
		ok = true
		return tx.Delete(&model.Pin{ID: id}).Error
	})
	if err != nil {
		return err
	}
	if ok {
		repo.hub.Publish(hub.Message{
			Name: event.MessageUnpinned,
			Fields: hub.Fields{
				"pin_id":     id,
				"message_id": pin.MessageID,
			},
		})
	}
	return nil
}

// GetPinsByChannelID チャンネルのピン留めを全て取得する
func (repo *RepositoryImpl) GetPinsByChannelID(channelID uuid.UUID) (pins []*model.Pin, err error) {
	pins = make([]*model.Pin, 0)
	if channelID == uuid.Nil {
		return pins, nil
	}
	err = repo.db.
		Joins("INNER JOIN messages ON messages.id = pins.message_id AND messages.channel_id = ?", channelID).
		Preload("Message").
		Find(&pins).
		Error
	return
}
