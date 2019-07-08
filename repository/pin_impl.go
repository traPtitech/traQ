package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"time"
)

// CreatePin implements PinRepository interface.
func (repo *GormRepository) CreatePin(messageID, userID uuid.UUID) (uuid.UUID, error) {
	if messageID == uuid.Nil || userID == uuid.Nil {
		return uuid.Nil, ErrNilID
	}
	var (
		p       model.Pin
		m       model.Message
		changed bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.First(&m, &model.Message{ID: messageID}).Error; err != nil {
			return convertError(err)
		}

		if err := tx.First(&p, &model.Pin{MessageID: messageID}).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				p = model.Pin{ID: uuid.Must(uuid.NewV4()), MessageID: messageID, UserID: userID}
				changed = true
				return tx.Create(&p).Error
			}
			return err
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}

	if changed {
		repo.hub.Publish(hub.Message{
			Name: event.MessagePinned,
			Fields: hub.Fields{
				"message_id": messageID,
				"pin_id":     p.ID,
			},
		})

		// ロギング
		go repo.recordChannelEvent(m.ChannelID, model.ChannelEventPinAdded, model.ChannelEventDetail{
			"userId":    userID,
			"messageId": messageID,
		}, p.CreatedAt)
	}

	return p.ID, err
}

// GetPin implements PinRepository interface.
func (repo *GormRepository) GetPin(id uuid.UUID) (p *model.Pin, err error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	p = &model.Pin{}
	err = repo.db.Scopes(pinPreloads).Where(&model.Pin{ID: id}).Take(p).Error
	if err != nil {
		return nil, convertError(err)
	}
	return p, nil
}

// IsPinned implements PinRepository interface.
func (repo *GormRepository) IsPinned(messageID uuid.UUID) (bool, error) {
	if messageID == uuid.Nil {
		return false, nil
	}
	return dbExists(repo.db, &model.Pin{MessageID: messageID})
}

// DeletePin implements PinRepository interface.
func (repo *GormRepository) DeletePin(pinID, userID uuid.UUID) error {
	if pinID == uuid.Nil {
		return ErrNilID
	}
	var (
		pin model.Pin
		ok  bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Preload("Message").Where(&model.Pin{ID: pinID}).First(&pin).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return nil
			}
			return err
		}
		ok = true
		return tx.Delete(&model.Pin{ID: pinID}).Error
	})
	if err != nil {
		return err
	}
	if ok {
		repo.hub.Publish(hub.Message{
			Name: event.MessageUnpinned,
			Fields: hub.Fields{
				"pin_id":     pinID,
				"message_id": pin.MessageID,
			},
		})

		// ロギング
		go repo.recordChannelEvent(pin.Message.ChannelID, model.ChannelEventPinRemoved, model.ChannelEventDetail{
			"userId":    userID,
			"messageId": pin.MessageID,
		}, time.Now())
	}
	return nil
}

// GetPinsByChannelID implements PinRepository interface.
func (repo *GormRepository) GetPinsByChannelID(channelID uuid.UUID) (pins []*model.Pin, err error) {
	pins = make([]*model.Pin, 0)
	if channelID == uuid.Nil {
		return pins, nil
	}
	err = repo.db.
		Scopes(pinPreloads).
		Joins("INNER JOIN messages ON messages.id = pins.message_id AND messages.channel_id = ?", channelID).
		Find(&pins).
		Error
	return
}

func pinPreloads(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Message").
		Preload("Message.Stamps", func(db *gorm.DB) *gorm.DB {
			return db.Order("updated_at")
		})
}
