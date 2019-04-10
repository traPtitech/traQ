package impl

import (
	"errors"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/message"
	"strings"
)

// CreateMessage メッセージを作成します
func (repo *RepositoryImpl) CreateMessage(userID, channelID uuid.UUID, text string) (*model.Message, error) {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	m := &model.Message{
		ID:        uuid.Must(uuid.NewV4()),
		UserID:    userID,
		ChannelID: channelID,
		Text:      text,
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	if err := repo.db.Create(m).Error; err != nil {
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
	return m, nil
}

// UpdateMessage メッセージを更新します
func (repo *RepositoryImpl) UpdateMessage(messageID uuid.UUID, text string) error {
	if messageID == uuid.Nil {
		return repository.ErrNilID
	}
	if len(text) == 0 {
		return errors.New("text is empty")
	}

	var (
		old model.Message
		new model.Message
		ok  bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&model.Message{ID: messageID}).First(&old).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return repository.ErrNotFound
			}
			return err
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

// DeleteMessage メッセージを削除します
func (repo *RepositoryImpl) DeleteMessage(messageID uuid.UUID) error {
	if messageID == uuid.Nil {
		return repository.ErrNilID
	}

	var (
		m  model.Message
		ok bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&model.Message{ID: messageID}).First(&m).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return repository.ErrNotFound
			}
			return err
		}

		if err := tx.Delete(&m).Error; err != nil {
			return err
		}
		if err := tx.Where(&model.Unread{MessageID: messageID}).Delete(model.Unread{}).Error; err != nil {
			return err
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

// GetMessageByID messageIDで指定されたメッセージを取得します
func (repo *RepositoryImpl) GetMessageByID(messageID uuid.UUID) (*model.Message, error) {
	if messageID == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	message := &model.Message{}
	if err := repo.db.Where(&model.Message{ID: messageID}).Take(message).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return message, nil
}

// GetMessagesByChannelID 指定されたチャンネルのメッセージを取得します
func (repo *RepositoryImpl) GetMessagesByChannelID(channelID uuid.UUID, limit, offset int) (arr []*model.Message, err error) {
	arr = make([]*model.Message, 0)
	if channelID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.
		Where(&model.Message{ChannelID: channelID}).
		Order("created_at DESC").
		Scopes(limitAndOffset(limit, offset)).
		Find(&arr).
		Error
	return arr, err
}

// GetMessagesByChannelID 指定されたユーザーのメッセージを取得します
func (repo *RepositoryImpl) GetMessagesByUserID(userID uuid.UUID, limit, offset int) (arr []*model.Message, err error) {
	arr = make([]*model.Message, 0)
	if userID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.
		Where(&model.Message{UserID: userID}).
		Order("created_at DESC").
		Scopes(limitAndOffset(limit, offset)).
		Find(&arr).
		Error
	return arr, err
}

// SetMessageUnread 指定したメッセージを未読にします
func (repo *RepositoryImpl) SetMessageUnread(userID, messageID uuid.UUID) error {
	if userID == uuid.Nil || messageID == uuid.Nil {
		return repository.ErrNilID
	}
	var u model.Unread
	return repo.db.FirstOrCreate(&u, &model.Unread{UserID: userID, MessageID: messageID}).Error
}

// GetUnreadMessagesByUserID あるユーザーの未読メッセージをすべて取得
func (repo *RepositoryImpl) GetUnreadMessagesByUserID(userID uuid.UUID) (unreads []*model.Message, err error) {
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

// DeleteUnreadsByMessageID 指定したメッセージIDの未読レコードを全て削除
func (repo *RepositoryImpl) DeleteUnreadsByMessageID(messageID uuid.UUID) error {
	if messageID == uuid.Nil {
		return repository.ErrNilID
	}
	return repo.db.Where(&model.Unread{MessageID: messageID}).Delete(model.Unread{}).Error
}

// DeleteUnreadsByChannelID 指定したチャンネルIDに存在する、指定したユーザーIDの未読レコードをすべて削除
func (repo *RepositoryImpl) DeleteUnreadsByChannelID(channelID, userID uuid.UUID) error {
	if channelID == uuid.Nil || userID == uuid.Nil {
		return repository.ErrNilID
	}
	result := repo.db.Exec("DELETE unreads FROM unreads INNER JOIN messages ON unreads.user_id = ? AND unreads.message_id = messages.id WHERE messages.channel_id = ?", userID.String(), channelID.String())
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

// GetChannelLatestMessagesByUserID 指定したユーザーが閲覧可能な全てのチャンネルの最新のメッセージの一覧を取得します
func (repo *RepositoryImpl) GetChannelLatestMessagesByUserID(userID uuid.UUID, limit int, subscribeOnly bool) ([]*model.Message, error) {
	var query string
	switch {
	case subscribeOnly:
		query = `
SELECT m.id, m.user_id, m.channel_id, m.text, m.created_at, m.updated_at, m.deleted_at
FROM (
       SELECT ROW_NUMBER() OVER(PARTITION BY m.channel_id ORDER BY m.created_at DESC) AS r,
              m.*
       FROM messages m
       WHERE m.deleted_at IS NULL
     ) m
       INNER JOIN channels c ON m.channel_id = c.id
       INNER JOIN (SELECT channel_id
                   FROM users_subscribe_channels
                   WHERE user_id = 'USER_ID'
                   UNION
                   SELECT channel_id
                   FROM users_private_channels
                   WHERE user_id = 'USER_ID') s ON s.channel_id = m.channel_id
WHERE m.r = 1 AND c.deleted_at IS NULL
ORDER BY m.created_at DESC
`
	default:
		query = `
SELECT m.id, m.user_id, m.channel_id, m.text, m.created_at, m.updated_at, m.deleted_at
FROM (
       SELECT ROW_NUMBER() OVER(PARTITION BY m.channel_id ORDER BY m.created_at DESC) AS r,
              m.*
       FROM messages m
       WHERE m.deleted_at IS NULL
     ) m
       INNER JOIN channels c ON m.channel_id = c.id
       LEFT JOIN users_private_channels upc ON upc.channel_id = m.channel_id
WHERE m.r = 1 AND c.deleted_at IS NULL AND (c.is_public = true OR upc.user_id = 'USER_ID')
ORDER BY m.created_at DESC
`
	}

	query = strings.Replace(query, "USER_ID", userID.String(), -1)
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	result := make([]*model.Message, 0)
	err := repo.db.Raw(query).Scan(&result).Error
	return result, err
}

// GetArchivedMessagesByID アーカイブメッセージを取得します
func (repo *RepositoryImpl) GetArchivedMessagesByID(messageID uuid.UUID) ([]*model.ArchivedMessage, error) {
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
