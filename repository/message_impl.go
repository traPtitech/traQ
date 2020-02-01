package repository

import (
	"fmt"
	"github.com/traPtitech/traQ/utils/message"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
)

// CreateMessage implements MessageRepository interface.
func (repo *GormRepository) CreateMessage(userID, channelID uuid.UUID, text string) (*model.Message, error) {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return nil, ErrNilID
	}

	m := &model.Message{
		ID:        uuid.Must(uuid.NewV4()),
		UserID:    userID,
		ChannelID: channelID,
		Text:      text,
		Stamps:    []model.MessageStamp{},
	}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(m).Error; err != nil {
			return err
		}

		clm := &model.ChannelLatestMessage{
			ChannelID: m.ChannelID,
			MessageID: m.ID,
			DateTime:  m.CreatedAt,
		}
		r := tx.Model(&model.ChannelLatestMessage{ChannelID: channelID}).Updates(clm)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			return tx.Create(clm).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	embedded, plain := message.Parse(text)
	repo.hub.Publish(hub.Message{
		Name: event.MessageCreated,
		Fields: hub.Fields{
			"message_id": m.ID,
			"message":    m,
			"embedded":   embedded,
			"plain":      plain,
		},
	})
	messagesCounter.Inc()
	return m, nil
}

// UpdateMessage implements MessageRepository interface.
func (repo *GormRepository) UpdateMessage(messageID uuid.UUID, text string) error {
	if messageID == uuid.Nil {
		return ErrNilID
	}

	var (
		old model.Message
		new model.Message
		ok  bool
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&old, &model.Message{ID: messageID}).Error; err != nil {
			return convertError(err)
		}

		// archiving
		if err := tx.Create(&model.ArchivedMessage{
			ID:        uuid.Must(uuid.NewV4()),
			MessageID: old.ID,
			UserID:    old.UserID,
			Text:      old.Text,
			DateTime:  old.UpdatedAt,
		}).Error; err != nil {
			return err
		}

		// update
		if err := tx.Model(&old).Update("text", text).Error; err != nil {
			return err
		}

		ok = true
		return tx.Where(&model.Message{ID: messageID}).First(&new).Error
	})
	if err != nil {
		return err
	}
	if ok {
		repo.hub.Publish(hub.Message{
			Name: event.MessageUpdated,
			Fields: hub.Fields{
				"message_id":  messageID,
				"old_message": &old,
				"message":     &new,
			},
		})
	}
	return nil
}

// DeleteMessage implements MessageRepository interface.
func (repo *GormRepository) DeleteMessage(messageID uuid.UUID) error {
	if messageID == uuid.Nil {
		return ErrNilID
	}

	var (
		m  model.Message
		ok bool
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(&model.Message{ID: messageID}).First(&m).Error; err != nil {
			return convertError(err)
		}
		errs := tx.
			Delete(&m).
			Delete(model.Unread{}, &model.Unread{MessageID: messageID}).
			Delete(model.Pin{}, &model.Pin{MessageID: messageID}).
			GetErrors()
		if len(errs) > 0 {
			return errs[0]
		}
		ok = true
		return nil
	})
	if err != nil {
		return err
	}
	if ok {
		repo.hub.Publish(hub.Message{
			Name: event.MessageDeleted,
			Fields: hub.Fields{
				"message_id": messageID,
				"message":    &m,
			},
		})
	}
	return nil
}

// GetMessageByID implements MessageRepository interface.
func (repo *GormRepository) GetMessageByID(messageID uuid.UUID) (*model.Message, error) {
	if messageID == uuid.Nil {
		return nil, ErrNotFound
	}
	message := &model.Message{}
	if err := repo.db.Scopes(messagePreloads).Where(&model.Message{ID: messageID}).Take(message).Error; err != nil {
		return nil, convertError(err)
	}
	return message, nil
}

// GetMessages implements MessageRepository interface.
func (repo *GormRepository) GetMessages(query MessagesQuery) (messages []*model.Message, more bool, err error) {
	messages = make([]*model.Message, 0)

	tx := repo.db.Scopes(messagePreloads)
	if query.Asc {
		tx = tx.Order("created_at")
	} else {
		tx = tx.Order("created_at DESC")
	}

	if query.Channel != uuid.Nil {
		tx = tx.Where("channel_id = ?", query.Channel)
	}
	if query.User != uuid.Nil {
		tx = tx.Where("user_id = ?", query.User)
	}

	if query.Inclusive {
		if query.Since.Valid {
			tx = tx.Where("created_at >= ?", query.Since.Time)
		}
		if query.Until.Valid {
			tx = tx.Where("created_at <= ?", query.Until.Time)
		}
	} else {
		if query.Since.Valid {
			tx = tx.Where("created_at > ?", query.Since.Time)
		}
		if query.Until.Valid {
			tx = tx.Where("created_at < ?", query.Until.Time)
		}
	}

	if query.Offset > 0 {
		tx = tx.Offset(query.Offset)
	}

	if query.Limit > 0 {
		err = tx.Limit(query.Limit + 1).Find(&messages).Error
		if len(messages) > query.Limit {
			return messages[:len(messages)-1], true, err
		}
	} else {
		err = tx.Find(&messages).Error
	}
	return messages, false, err
}

// SetMessageUnread implements MessageRepository interface.
func (repo *GormRepository) SetMessageUnread(userID, messageID uuid.UUID, noticeable bool) error {
	if userID == uuid.Nil || messageID == uuid.Nil {
		return ErrNilID
	}
	var u model.Unread
	return repo.db.Assign(map[string]bool{"noticeable": noticeable}).FirstOrCreate(&u, &model.Unread{UserID: userID, MessageID: messageID}).Error
}

// GetUnreadMessagesByUserID implements MessageRepository interface.
func (repo *GormRepository) GetUnreadMessagesByUserID(userID uuid.UUID) (unreads []*model.Message, err error) {
	unreads = make([]*model.Message, 0)
	if userID == uuid.Nil {
		return unreads, nil
	}
	err = repo.db.
		Joins("INNER JOIN unreads ON unreads.message_id = messages.id AND unreads.user_id = ?", userID.String()).
		Order("messages.created_at").
		Find(&unreads).
		Error
	return unreads, err
}

// GetUserUnreadChannels implements MessageRepository interface.
func (repo *GormRepository) GetUserUnreadChannels(userID uuid.UUID) ([]*UserUnreadChannel, error) {
	res := make([]*UserUnreadChannel, 0)
	if userID == uuid.Nil {
		return res, nil
	}
	return res, repo.db.Raw(`SELECT m.channel_id AS channel_id, COUNT(m.id) AS count, MAX(u.noticeable) AS noticeable, MIN(m.created_at) AS since, MAX(m.created_at) AS updated_at FROM unreads u JOIN messages m on u.message_id = m.id WHERE u.user_id = ? GROUP BY m.channel_id`, userID).Scan(&res).Error
}

// DeleteUnreadsByChannelID implements MessageRepository interface.
func (repo *GormRepository) DeleteUnreadsByChannelID(channelID, userID uuid.UUID) error {
	if channelID == uuid.Nil || userID == uuid.Nil {
		return ErrNilID
	}
	result := repo.db.Exec("DELETE unreads FROM unreads INNER JOIN messages ON unreads.user_id = ? AND unreads.message_id = messages.id WHERE messages.channel_id = ?", userID, channelID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.ChannelRead,
			Fields: hub.Fields{
				"channel_id": channelID,
				"user_id":    userID,
			},
		})
	}
	return nil
}

// GetChannelLatestMessagesByUserID implements MessageRepository interface.
func (repo *GormRepository) GetChannelLatestMessagesByUserID(userID uuid.UUID, limit int, subscribeOnly bool) ([]*model.Message, error) {
	var query string
	switch {
	case subscribeOnly:
		query = `SELECT m.id, m.user_id, m.channel_id, m.text, m.created_at, m.updated_at, m.deleted_at FROM channel_latest_messages clm INNER JOIN messages m ON clm.message_id = m.id INNER JOIN channels c ON clm.channel_id = c.id WHERE c.deleted_at IS NULL AND c.is_public = TRUE AND m.deleted_at IS NULL AND (c.is_forced = TRUE OR c.id IN (SELECT s.channel_id FROM users_subscribe_channels s WHERE s.user_id = 'USER_ID')) ORDER BY clm.date_time DESC`
		query = strings.Replace(query, "USER_ID", userID.String(), -1)
	default:
		query = `SELECT m.id, m.user_id, m.channel_id, m.text, m.created_at, m.updated_at, m.deleted_at FROM channel_latest_messages clm INNER JOIN messages m ON clm.message_id = m.id INNER JOIN channels c ON clm.channel_id = c.id WHERE c.deleted_at IS NULL AND c.is_public = TRUE AND m.deleted_at IS NULL ORDER BY clm.date_time DESC`
	}

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	result := make([]*model.Message, 0)
	return result, repo.db.Raw(query).Scan(&result).Error
}

// GetArchivedMessagesByID implements MessageRepository interface.
func (repo *GormRepository) GetArchivedMessagesByID(messageID uuid.UUID) ([]*model.ArchivedMessage, error) {
	r := make([]*model.ArchivedMessage, 0)
	if messageID == uuid.Nil {
		return r, nil
	}
	err := repo.db.
		Where(&model.ArchivedMessage{MessageID: messageID}).
		Order("date_time").
		Find(&r).
		Error
	return r, err
}

func messagePreloads(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Stamps", func(db *gorm.DB) *gorm.DB {
			return db.Order("updated_at")
		}).
		Preload("Pin")
}
