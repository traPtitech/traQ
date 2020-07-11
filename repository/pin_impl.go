package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"time"
)

// PinMessage implements PinRepository interface.
func (repo *GormRepository) PinMessage(messageID, userID uuid.UUID) (*model.Pin, error) {
	if messageID == uuid.Nil || userID == uuid.Nil {
		return nil, ErrNilID
	}
	var (
		p       model.Pin
		m       model.Message
		changed bool
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
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
		return nil, err
	}

	if changed {
		repo.hub.Publish(hub.Message{
			Name: event.MessagePinned,
			Fields: hub.Fields{
				"message_id": messageID,
				"channel_id": m.ChannelID,
			},
		})

		// ロギング
		go repo.recordChannelEvent(m.ChannelID, model.ChannelEventPinAdded, model.ChannelEventDetail{
			"userId":    userID,
			"messageId": messageID,
		}, p.CreatedAt)
	}

	return &p, err
}

// UnpinMessage implements PinRepository interface.
func (repo *GormRepository) UnpinMessage(messageID, userID uuid.UUID) error {
	if messageID == uuid.Nil {
		return ErrNilID
	}
	var (
		pin model.Pin
		ok  bool
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Message").Where(&model.Pin{MessageID: messageID}).First(&pin).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return nil
			}
			return err
		}
		ok = true
		return tx.Delete(model.Pin{}, &model.Pin{MessageID: messageID}).Error
	})
	if err != nil {
		return err
	}
	if ok {
		repo.hub.Publish(hub.Message{
			Name: event.MessageUnpinned,
			Fields: hub.Fields{
				"channel_id": pin.Message.ChannelID,
				"message_id": messageID,
			},
		})

		// ロギング
		go repo.recordChannelEvent(pin.Message.ChannelID, model.ChannelEventPinRemoved, model.ChannelEventDetail{
			"userId":    userID,
			"messageId": messageID,
		}, time.Now())
	}
	return nil
}

// GetPinnedMessageByChannelID implements PinRepository interface.
func (repo *GormRepository) GetPinnedMessageByChannelID(channelID uuid.UUID) (pins []*model.Pin, err error) {
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
		Preload("Message.Stamps")
}
