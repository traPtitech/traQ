package gorm

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/message"
)

// CreateMessage implements MessageRepository interface.
func (repo *Repository) CreateMessage(userID, channelID uuid.UUID, text string) (*model.Message, error) {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return nil, repository.ErrNilID
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

		return tx.
			Clauses(clause.OnConflict{UpdateAll: true}).
			Create(clm).
			Error
	})
	if err != nil {
		return nil, err
	}

	parseResult := message.Parse(text)
	repo.hub.Publish(hub.Message{
		Name: event.MessageCreated,
		Fields: hub.Fields{
			"message_id":   m.ID,
			"message":      m,
			"parse_result": parseResult,
		},
	})
	if len(parseResult.Citation) > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.MessageCited,
			Fields: hub.Fields{
				"message_id": m.ID,
				"message":    m,
				"cited_ids":  parseResult.Citation,
			},
		})
	}
	return m, nil
}

// UpdateMessage implements MessageRepository interface.
func (repo *Repository) UpdateMessage(messageID uuid.UUID, text string) error {
	if messageID == uuid.Nil {
		return repository.ErrNilID
	}

	var (
		oldMes model.Message
		newMes model.Message
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&oldMes, &model.Message{ID: messageID}).Error; err != nil {
			return convertError(err)
		}

		// archiving
		if err := tx.Create(&model.ArchivedMessage{
			ID:        uuid.Must(uuid.NewV4()),
			MessageID: oldMes.ID,
			UserID:    oldMes.UserID,
			Text:      oldMes.Text,
			DateTime:  oldMes.UpdatedAt,
		}).Error; err != nil {
			return err
		}

		// update
		if err := tx.Model(&oldMes).Update("text", text).Error; err != nil {
			return err
		}

		return tx.Where(&model.Message{ID: messageID}).First(&newMes).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.MessageUpdated,
		Fields: hub.Fields{
			"message_id":  messageID,
			"old_message": &oldMes,
			"message":     &newMes,
		},
	})
	return nil
}

// DeleteMessage implements MessageRepository interface.
func (repo *Repository) DeleteMessage(messageID uuid.UUID) error {
	if messageID == uuid.Nil {
		return repository.ErrNilID
	}

	var (
		m       model.Message
		unreads []*model.Unread
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(&model.Message{ID: messageID}).First(&m).Error; err != nil {
			return convertError(err)
		}

		if err := tx.Find(&unreads, &model.Unread{MessageID: messageID}).Error; err != nil {
			return err
		}

		if err := tx.Delete(&m).Error; err != nil {
			return err
		}
		if err := tx.Delete(model.Unread{}, &model.Unread{MessageID: messageID}).Error; err != nil {
			return err
		}
		if err := tx.Delete(model.Pin{}, &model.Pin{MessageID: messageID}).Error; err != nil {
			return err
		}
		if err := tx.Delete(model.ClipFolderMessage{}, &model.ClipFolderMessage{MessageID: messageID}).Error; err != nil {
			return err
		}

		var mes []model.Message
		if err := tx.
			Where(&model.Message{ChannelID: m.ChannelID}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: "created_at"}, Desc: true}).
			Limit(1).
			Find(&mes).Error; err != nil {
			return err
		}
		if len(mes) != 1 {
			return nil
		}
		return tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&model.ChannelLatestMessage{
			ChannelID: mes[0].ChannelID,
			MessageID: mes[0].ID,
			DateTime:  mes[0].CreatedAt,
		}).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.MessageDeleted,
		Fields: hub.Fields{
			"message_id":      messageID,
			"message":         &m,
			"deleted_unreads": unreads,
		},
	})
	return nil
}

// GetMessageByID implements MessageRepository interface.
func (repo *Repository) GetMessageByID(messageID uuid.UUID) (*model.Message, error) {
	if messageID == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	message := &model.Message{}
	if err := repo.db.Scopes(messagePreloads).Where(&model.Message{ID: messageID}).Take(message).Error; err != nil {
		return nil, convertError(err)
	}
	return message, nil
}

// GetMessages implements MessageRepository interface.
func (repo *Repository) GetMessages(query repository.MessagesQuery) (messages []*model.Message, more bool, err error) {
	messages = make([]*model.Message, 0)

	tx := repo.db
	if !query.DisablePreload {
		tx = tx.Scopes(messagePreloads)
	}

	if query.Asc {
		tx = tx.Order("messages.created_at")
	} else {
		tx = tx.Order("messages.created_at DESC")
	}

	if query.Offset > 0 {
		tx = tx.Offset(query.Offset)
	}

	if query.ChannelsSubscribedByUser != uuid.Nil || query.ExcludeDMs {
		// JOIN時にidx_messages_channel_id_deleted_at_created_atが使われてしまい、JOIN、WHERE後のレコード数が多い場合はfile sortで重くなる
		// NOTE: gorm.io/hints はJOINと同時に使うと順番が壊れる
		tx = tx.Joins("USE INDEX (`idx_messages_deleted_at_created_at`) INNER JOIN channels ON messages.channel_id = channels.id")
	}

	if query.IDIn.Valid {
		tx = tx.Where("messages.id IN ?", query.IDIn.V)
	}
	if query.Channel != uuid.Nil {
		tx = tx.Where("messages.channel_id = ?", query.Channel)
	}
	if query.User != uuid.Nil {
		tx = tx.Where("messages.user_id = ?", query.User)
	}
	if query.ChannelsSubscribedByUser != uuid.Nil {
		tx = tx.Where("channels.is_forced = TRUE OR channels.id IN (SELECT s.channel_id FROM users_subscribe_channels s WHERE s.user_id = ?)", query.ChannelsSubscribedByUser)
	}

	if query.Inclusive {
		if query.Since.Valid {
			tx = tx.Where("messages.created_at >= ?", query.Since.V.Truncate(time.Microsecond))
		}
		if query.Until.Valid {
			tx = tx.Where("messages.created_at <= ?", query.Until.V.Truncate(time.Microsecond))
		}
	} else {
		if query.Since.Valid {
			tx = tx.Where("messages.created_at > ?", query.Since.V.Truncate(time.Microsecond))
		}
		if query.Until.Valid {
			tx = tx.Where("messages.created_at < ?", query.Until.V.Truncate(time.Microsecond))
		}
	}

	if query.ExcludeDMs {
		tx = tx.Where("channels.is_public = true")
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

// GetUpdatedMessagesAfter implements MessageRepository interface.
func (repo *Repository) GetUpdatedMessagesAfter(after time.Time, limit int) (messages []*model.Message, more bool, err error) {
	err = repo.db.
		Raw("SELECT * FROM `messages` USE INDEX (idx_messages_deleted_at_updated_at) WHERE `messages`.`deleted_at` IS NULL AND `messages`.`updated_at` > ? ORDER BY `messages`.`updated_at` LIMIT ?", after, limit+1).
		Scan(&messages).
		Error

	if len(messages) > limit {
		more = true
		messages = messages[:limit]
	}
	return
}

// GetDeletedMessagesAfter implements MessageRepository interface.
func (repo *Repository) GetDeletedMessagesAfter(after time.Time, limit int) (messages []*model.Message, more bool, err error) {
	err = repo.db.
		Raw("SELECT * FROM `messages` USE INDEX (idx_messages_deleted_at_updated_at) WHERE `messages`.`deleted_at` > ? ORDER BY `messages`.`deleted_at` LIMIT ?", after, limit+1).
		Scan(&messages).
		Error

	if len(messages) > limit {
		more = true
		messages = messages[:limit]
	}
	return
}

// SetMessageUnread implements MessageRepository interface.
func (repo *Repository) SetMessageUnread(userID, messageID uuid.UUID, noticeable bool) error {
	if userID == uuid.Nil || messageID == uuid.Nil {
		return repository.ErrNilID
	}

	var update bool
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var u model.Unread
		if err := tx.First(&u, &model.Unread{UserID: userID, MessageID: messageID}).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				var m model.Message
				err := tx.First(&m, &model.Message{ID: messageID}).Error
				if err != nil {
					return err
				}

				return tx.Create(&model.Unread{
					UserID:           userID,
					ChannelID:        m.ChannelID,
					MessageID:        messageID,
					Noticeable:       noticeable,
					MessageCreatedAt: m.CreatedAt,
				}).Error
			}
			return err
		}
		update = true
		return tx.Model(&u).Update("noticeable", noticeable).Error
	})
	if err != nil {
		return err
	}
	if !update {
		repo.hub.Publish(hub.Message{
			Name: event.MessageUnread,
			Fields: hub.Fields{
				"message_id": messageID,
				"user_id":    userID,
				"noticeable": noticeable,
			},
		})
	}
	return nil
}

// GetUnreadMessagesByUserID implements MessageRepository interface.
func (repo *Repository) GetUnreadMessagesByUserID(userID uuid.UUID) (unreads []*model.Message, err error) {
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
func (repo *Repository) GetUserUnreadChannels(userID uuid.UUID) ([]*repository.UserUnreadChannel, error) {
	res := make([]*repository.UserUnreadChannel, 0)
	if userID == uuid.Nil {
		return res, nil
	}
	return res, repo.db.Raw(`
		SELECT
			channel_id,
			COUNT(message_id) AS count,
			MAX(noticeable) AS noticeable,
			MIN(message_created_at) AS since,
			MAX(message_created_at) AS updated_at,
			(
				SELECT message_id
				FROM unreads
				WHERE user_id = MIN(u.user_id)
					AND channel_id = u.channel_id
					AND message_created_at = MIN(u.message_created_at)
				LIMIT 1
			) AS oldest_message_id
		/*
			2023/04/26時点
			oldest_message_id 取得部分はサブクエリがN+1で叩かれることになるが、
			これ以外の方法(2つのクエリに分けて取得など)では大規模なJOINが発生
			してしまいパフォーマンスが悪くなるため、この方法を使う。
		*/
		FROM unreads u
		WHERE user_id = ?
		GROUP BY channel_id;
	`, userID).Scan(&res).Error
}

// DeleteUnreadsByChannelID implements MessageRepository interface.
func (repo *Repository) DeleteUnreadsByChannelID(channelID, userID uuid.UUID) error {
	if channelID == uuid.Nil || userID == uuid.Nil {
		return repository.ErrNilID
	}
	result := repo.db.Where("user_id = ?", userID).Where("channel_id = ?", channelID).Delete(&model.Unread{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.ChannelRead,
			Fields: hub.Fields{
				"channel_id":        channelID,
				"user_id":           userID,
				"read_messages_num": int(result.RowsAffected),
			},
		})
	}
	return nil
}

// GetChannelLatestMessages implements MessageRepository interface.
func (repo *Repository) GetChannelLatestMessages(query repository.ChannelLatestMessagesQuery) ([]*model.Message, error) {
	var messages []*model.Message

	tx := repo.db.
		Unscoped().
		Select("m.id, m.user_id, m.channel_id, m.text, m.created_at, m.updated_at, m.deleted_at").
		Table("channel_latest_messages clm").
		Joins("INNER JOIN messages m ON clm.message_id = m.id").
		Joins("INNER JOIN channels c ON clm.channel_id = c.id").
		Where("c.deleted_at IS NULL AND c.is_public = TRUE AND m.deleted_at IS NULL").
		Order("clm.date_time DESC")

	if query.SubscribedByUser.Valid {
		tx = tx.Where("c.is_forced = TRUE OR c.id IN (?)", repo.db.Table("users_subscribe_channels").Select("channel_id").Where("user_id", query.SubscribedByUser.V))
	}

	if query.Limit > 0 {
		tx = tx.Limit(query.Limit)
	}

	if query.Since.Valid {
		tx = tx.Where("clm.date_time >= ?", query.Since.V)
	}

	return messages, tx.Find(&messages).Error
}

// AddStampToMessage implements MessageRepository interface.
func (repo *Repository) AddStampToMessage(messageID, stampID, userID uuid.UUID, count int) (ms *model.MessageStamp, err error) {
	if messageID == uuid.Nil || stampID == uuid.Nil || userID == uuid.Nil {
		return nil, repository.ErrNilID
	}

	err = repo.db.
		Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(map[string]interface{}{
				"count":      gorm.Expr(fmt.Sprintf("count + %d", count)),
				"updated_at": gorm.Expr("now()"),
			}),
		}).
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

// RemoveStampFromMessage implements MessageRepository interface.
func (repo *Repository) RemoveStampFromMessage(messageID, stampID, userID uuid.UUID) (err error) {
	if messageID == uuid.Nil || stampID == uuid.Nil || userID == uuid.Nil {
		return repository.ErrNilID
	}
	result := repo.db.Delete(&model.MessageStamp{}, &model.MessageStamp{MessageID: messageID, StampID: stampID, UserID: userID})
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

func messagePreloads(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Stamps").
		Preload("Pin")
}
