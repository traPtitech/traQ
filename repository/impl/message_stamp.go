package impl

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

// AddStampToMessage メッセージにスタンプを追加します
func (repo *RepositoryImpl) AddStampToMessage(messageID, stampID, userID uuid.UUID) (ms *model.MessageStamp, err error) {
	if messageID == uuid.Nil || stampID == uuid.Nil || userID == uuid.Nil {
		return nil, repository.ErrNilID
	}

	err = repo.db.
		Set("gorm:insert_option", "ON DUPLICATE KEY UPDATE count = count + 1, updated_at = now()").
		Create(&model.MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID, Count: 1}).
		Error
	if err != nil {
		return nil, err
	}

	ms = &model.MessageStamp{}
	if err := repo.db.Where(&model.MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID}).Take(ms).Error; err != nil {
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

// RemoveStampFromMessage メッセージからスタンプを削除します
func (repo *RepositoryImpl) RemoveStampFromMessage(messageID, stampID, userID uuid.UUID) (err error) {
	if messageID == uuid.Nil || stampID == uuid.Nil || userID == uuid.Nil {
		return repository.ErrNilID
	}
	result := repo.db.Where(&model.MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID}).Delete(&model.MessageStamp{})
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

// GetMessageStamps メッセージのスタンプを取得します
func (repo *RepositoryImpl) GetMessageStamps(messageID uuid.UUID) (stamps []*model.MessageStamp, err error) {
	stamps = make([]*model.MessageStamp, 0)
	if messageID == uuid.Nil {
		return
	}
	err = repo.db.
		Joins("JOIN stamps ON messages_stamps.stamp_id = stamps.id AND messages_stamps.message_id = ?", messageID.String()).
		Order("messages_stamps.updated_at").
		Find(&stamps).
		Error
	return
}

// GetUserStampHistory ユーザーのスタンプ履歴を最大50件取得します。
func (repo *RepositoryImpl) GetUserStampHistory(userID uuid.UUID) (h []*model.UserStampHistory, err error) {
	h = make([]*model.UserStampHistory, 0)
	if userID == uuid.Nil {
		return
	}
	err = repo.db.
		Table("messages_stamps").
		Where("user_id = ?", userID.String()).
		Group("stamp_id").
		Select("stamp_id, max(updated_at) AS datetime").
		Order("datetime DESC").
		Limit(50).
		Scan(&h).
		Error
	return
}
