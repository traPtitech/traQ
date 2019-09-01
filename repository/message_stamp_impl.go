package repository

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
)

// AddStampToMessage implements MessageStampRepository interface.
func (repo *GormRepository) AddStampToMessage(messageID, stampID, userID uuid.UUID, count int) (ms *model.MessageStamp, err error) {
	if messageID == uuid.Nil || stampID == uuid.Nil || userID == uuid.Nil {
		return nil, ErrNilID
	}

	err = repo.db.
		Set("gorm:insert_option", fmt.Sprintf("ON DUPLICATE KEY UPDATE count = count + %d, updated_at = now()", count)).
		Create(&model.MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID, Count: count}).
		Error
	if err != nil {
		return nil, err
	}

	// 楽観的に取得し直す。
	ms = &model.MessageStamp{}
	if err := repo.db.Take(ms, &model.MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID}).Error; err != nil {
		return nil, err
	}
	repo.hub.Publish(hub.Message{
		Name: event.MessageStamped,
		Fields: hub.Fields{
			"message_id": messageID,
			"stamp_id":   stampID,
			"user_id":    userID,
			"count":      ms.Count,
			"created_at": ms.CreatedAt,
		},
	})
	return ms, nil
}

// RemoveStampFromMessage implements MessageStampRepository interface.
func (repo *GormRepository) RemoveStampFromMessage(messageID, stampID, userID uuid.UUID) (err error) {
	if messageID == uuid.Nil || stampID == uuid.Nil || userID == uuid.Nil {
		return ErrNilID
	}
	result := repo.db.Delete(&model.MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.MessageUnstamped,
			Fields: hub.Fields{
				"message_id": messageID,
				"stamp_id":   stampID,
				"user_id":    userID,
			},
		})
	}
	return nil
}

// GetMessageStamps implements MessageStampRepository interface.
func (repo *GormRepository) GetMessageStamps(messageID uuid.UUID) (stamps []*model.MessageStamp, err error) {
	stamps = make([]*model.MessageStamp, 0)
	if messageID == uuid.Nil {
		return
	}
	err = repo.db.
		Joins("JOIN stamps ON messages_stamps.stamp_id = stamps.id AND messages_stamps.message_id = ?", messageID).
		Order("messages_stamps.updated_at").
		Find(&stamps).
		Error
	return
}

// GetUserStampHistory implements MessageStampRepository interface.
func (repo *GormRepository) GetUserStampHistory(userID uuid.UUID) (h []*model.UserStampHistory, err error) {
	h = make([]*model.UserStampHistory, 0)
	if userID == uuid.Nil {
		return
	}
	err = repo.db.
		Table("messages_stamps").
		Where("user_id = ?", userID).
		Group("stamp_id").
		Select("stamp_id, max(updated_at) AS datetime").
		Order("datetime DESC").
		Limit(100).
		Scan(&h).
		Error
	return
}
