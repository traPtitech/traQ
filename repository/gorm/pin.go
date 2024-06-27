package gorm

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/gormutil"
)

// PinMessage implements PinRepository interface.
func (repo *Repository) PinMessage(messageID, userID uuid.UUID) (*model.Pin, error) {
	if messageID == uuid.Nil || userID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	var (
		p model.Pin
		m model.Message
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&m, &model.Message{ID: messageID}).Error; err != nil {
			return convertError(err)
		}

		p = model.Pin{ID: uuid.Must(uuid.NewV7()), MessageID: messageID, UserID: userID}
		if err := tx.Create(&p).Error; err != nil {
			if gormutil.IsMySQLDuplicatedRecordErr(err) {
				return repository.ErrAlreadyExists
			}
			return err
		}
		p.Message = m
		return nil
	})
	if err != nil {
		return nil, err
	}
	repo.hub.Publish(hub.Message{
		Name: event.MessagePinned,
		Fields: hub.Fields{
			"message_id": messageID,
			"channel_id": m.ChannelID,
		},
	})
	return &p, err
}

// UnpinMessage implements PinRepository interface.
func (repo *Repository) UnpinMessage(messageID uuid.UUID) (*model.Pin, error) {
	if messageID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	var pin model.Pin
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Preload("Message").Where(&model.Pin{MessageID: messageID}).First(&pin).Error; err != nil {
			return convertError(err)
		}
		return tx.Delete(model.Pin{}, &model.Pin{MessageID: messageID}).Error
	})
	if err != nil {
		return nil, err
	}
	repo.hub.Publish(hub.Message{
		Name: event.MessageUnpinned,
		Fields: hub.Fields{
			"channel_id": pin.Message.ChannelID,
			"message_id": messageID,
		},
	})
	return &pin, nil
}

// GetPinnedMessageByChannelID implements PinRepository interface.
func (repo *Repository) GetPinnedMessageByChannelID(channelID uuid.UUID) (pins []*model.Pin, err error) {
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
