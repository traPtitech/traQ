package gorm

import (
	"bytes"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/gormutil"
	"github.com/traPtitech/traQ/utils/set"
)

var dmChannelRootUUID = uuid.Must(uuid.FromString(model.DirectMessageChannelRootID))

// CreateChannel implements ChannelRepository interface.
func (repo *Repository) CreateChannel(ch model.Channel, privateMembers set.UUID, dm bool) (*model.Channel, error) {
	arr := []interface{}{&ch}

	ch.ID = uuid.Must(uuid.NewV7())
	ch.IsPublic = true
	ch.DeletedAt = gorm.DeletedAt{}

	if len(privateMembers) > 0 {
		ch.IsPublic = false
		ch.IsForced = false
		for uid := range privateMembers {
			arr = append(arr, &model.UsersPrivateChannel{
				UserID:    uid,
				ChannelID: ch.ID,
			})
		}
	}

	if dm {
		ch.ParentID = dmChannelRootUUID
		ch.IsPublic = false
		ch.IsForced = false

		m := &model.DMChannelMapping{
			ChannelID: ch.ID,
			User1:     uuid.UUID{},
			User2:     uuid.UUID{},
		}
		if l := len(privateMembers); l == 1 {
			users := privateMembers.Array()
			m.User1 = users[0]
			m.User2 = users[0]
		} else if l == 2 {
			users := privateMembers.Array()
			// user1 <= user2 になるように入れかえ
			if bytes.Compare(users[0].Bytes(), users[1].Bytes()) == 1 {
				t := users[0]
				users[0] = users[1]
				users[1] = t
			}

			m.User1 = users[0]
			m.User2 = users[1]
		} else {
			return nil, repository.ArgError("privateMembers", "length must be 1 or 2")
		}
		arr = append(arr, m)
	}

	err := repo.db.Transaction(func(tx *gorm.DB) error {
		for _, v := range arr {
			if err := tx.Create(v).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ChannelCreated,
		Fields: hub.Fields{
			"channel_id": ch.ID,
			"channel":    &ch,
			"private":    !ch.IsPublic,
		},
	})
	return &ch, nil
}

// UpdateChannel implements ChannelRepository interface.
func (repo *Repository) UpdateChannel(channelID uuid.UUID, args repository.UpdateChannelArgs) (*model.Channel, error) {
	if channelID == uuid.Nil {
		return nil, repository.ErrNilID
	}

	var ch model.Channel
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&ch, &model.Channel{ID: channelID}).Error; err != nil {
			return convertError(err)
		}

		data := map[string]interface{}{"updater_id": args.UpdaterID}
		if args.Topic.Valid {
			data["topic"] = args.Topic.V
		}
		if args.Visibility.Valid {
			data["is_visible"] = args.Visibility.V
		}
		if args.ForcedNotification.Valid {
			data["is_forced"] = args.ForcedNotification.V
		}
		if args.Name.Valid {
			data["name"] = args.Name.V
		}
		if args.Parent.Valid {
			data["parent_id"] = args.Parent.V
		}

		if err := tx.Model(&ch).Updates(data).Error; err != nil {
			return err
		}
		return tx.First(&ch, &model.Channel{ID: channelID}).Error
	})
	if err != nil {
		return nil, err
	}

	repo.hub.Publish(hub.Message{
		Name: event.ChannelUpdated,
		Fields: hub.Fields{
			"channel_id": channelID,
			"private":    !ch.IsPublic,
		},
	})
	if args.Topic.Valid {
		repo.hub.Publish(hub.Message{
			Name: event.ChannelTopicUpdated,
			Fields: hub.Fields{
				"channel_id": channelID,
				"topic":      args.Topic.V,
				"updater_id": args.UpdaterID,
			},
		})
	}
	return &ch, nil
}

// ArchiveChannels implements ChannelRepository interface.
func (repo *Repository) ArchiveChannels(ids []uuid.UUID) ([]*model.Channel, error) {
	var changed []*model.Channel
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			if id != uuid.Nil {
				var ch model.Channel
				if err := tx.First(&ch, &model.Channel{ID: id}).Error; err != nil {
					return err
				}
				if !ch.IsVisible {
					continue
				}
				if err := tx.Model(&ch).Updates(map[string]interface{}{"is_visible": false}).Error; err != nil {
					return err
				}
				if err := tx.First(&ch, &model.Channel{ID: id}).Error; err != nil {
					return err
				}
				changed = append(changed, &ch)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, ch := range changed {
		repo.hub.Publish(hub.Message{
			Name: event.ChannelUpdated,
			Fields: hub.Fields{
				"channel_id": ch.ID,
				"private":    !ch.IsPublic,
			},
		})
	}
	return changed, nil
}

// GetChannel implements ChannelRepository interface.
func (repo *Repository) GetChannel(channelID uuid.UUID) (*model.Channel, error) {
	if channelID == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	var ch model.Channel
	if err := repo.db.First(&ch, &model.Channel{ID: channelID}).Error; err != nil {
		return nil, convertError(err)
	}
	return &ch, nil
}

// GetPublicChannels implements ChannelRepository interface.
func (repo *Repository) GetPublicChannels() (channels []*model.Channel, err error) {
	channels = make([]*model.Channel, 0)
	return channels, repo.db.
		Where(&model.Channel{IsPublic: true}).
		Find(&channels).
		Error
}

// GetDirectMessageChannel implements ChannelRepository interface.
func (repo *Repository) GetDirectMessageChannel(user1, user2 uuid.UUID) (*model.Channel, error) {
	// user1 <= user2 になるように入れかえ
	if bytes.Compare(user1.Bytes(), user2.Bytes()) == 1 {
		t := user1
		user1 = user2
		user2 = t
	}

	// チャンネル存在確認
	var ch model.Channel
	err := repo.db.
		Where("id = (SELECT channel_id FROM dm_channel_mappings WHERE user1 = ? AND user2 = ?)", user1, user2).
		First(&ch).
		Error
	if err != nil {
		return nil, convertError(err)
	}
	return &ch, nil
}

// GetDirectMessageChannelMapping implements ChannelRepository interface.
func (repo *Repository) GetDirectMessageChannelMapping(userID uuid.UUID) (mappings []*model.DMChannelMapping, err error) {
	mappings = make([]*model.DMChannelMapping, 0)
	if userID == uuid.Nil {
		return
	}
	return mappings, repo.db.
		Where("user1 = ? OR user2 = ?", userID, userID).
		Find(&mappings).
		Error
}

// GetDirectMessageChannelList implements ChannelRepository interface.
func (repo *Repository) GetDirectMessageChannelList(userID uuid.UUID) (dmChannels []model.DMChannel, err error) {
	dmChannelMapping := make([]*model.DMChannelMapping, 0)
	if userID == uuid.Nil {
		return dmChannels, nil
	}

	if err := repo.db.
		Raw(`(SELECT dm.*, clm.date_time 
		      FROM dm_channel_mappings AS dm 
		      RIGHT JOIN channel_latest_messages AS clm ON dm.channel_id = clm.channel_id 
		      WHERE dm.user1 = ?)
		     UNION
		     (SELECT dm.*, clm.date_time 
		      FROM dm_channel_mappings AS dm 
		      RIGHT JOIN channel_latest_messages AS clm ON dm.channel_id = clm.channel_id 
		      WHERE dm.user2 = ?)
		     ORDER BY date_time DESC 
		     LIMIT 20`, userID, userID).
		Scan(&dmChannelMapping).
		Error; err != nil {
		return dmChannels, err
	}

	for _, mapping := range dmChannelMapping {
		var targetUserID uuid.UUID
		if mapping.User1 == userID {
			targetUserID = mapping.User2
		} else {
			targetUserID = mapping.User1
		}

		dmChannels = append(dmChannels, model.DMChannel{
			ID:     mapping.ChannelID,
			UserID: targetUserID,
		})
	}

	return dmChannels, nil
}

// GetPrivateChannelMemberIDs implements ChannelRepository interface.
func (repo *Repository) GetPrivateChannelMemberIDs(channelID uuid.UUID) (users []uuid.UUID, err error) {
	users = make([]uuid.UUID, 0)
	if channelID == uuid.Nil {
		return users, nil
	}
	return users, repo.db.
		Model(&model.UsersPrivateChannel{}).
		Where(&model.UsersPrivateChannel{ChannelID: channelID}).
		Pluck("user_id", &users).
		Error
}

// ChangeChannelSubscription implements ChannelRepository interface.
func (repo *Repository) ChangeChannelSubscription(channelID uuid.UUID, args repository.ChangeChannelSubscriptionArgs) (on []uuid.UUID, off []uuid.UUID, err error) {
	if channelID == uuid.Nil {
		return nil, nil, repository.ErrNilID
	}

	on = make([]uuid.UUID, 0)
	off = make([]uuid.UUID, 0)

	uids := make([]uuid.UUID, 0)
	for uid := range args.Subscription {
		uids = append(uids, uid)
	}

	err = repo.db.Transaction(func(tx *gorm.DB) error {
		// 指定された各ユーザーの現在のチャンネルの購読設定を取得
		var _current []*model.UserSubscribeChannel
		if err := tx.
			Where(&model.UserSubscribeChannel{ChannelID: channelID}).
			Where("user_id IN (?)", uids).
			Find(&_current).
			Error; err != nil {
			return err
		}
		current := make(map[uuid.UUID]model.ChannelSubscribeLevel, len(_current))
		for _, s := range _current {
			current[s.UserID] = s.GetLevel()
		}

		for uid, level := range args.Subscription {
			if cl := current[uid]; cl == level {
				continue // 既に同じ設定がされているのでスキップ
			}

			switch level {
			case model.ChannelSubscribeLevelNone:
				if _, ok := current[uid]; !ok {
					continue // 既にオフ
				}

				if args.KeepOffLevel {
					if cl := current[uid]; cl == model.ChannelSubscribeLevelMark {
						continue // 未読管理のみをキープしたままにする
					}
				}

				if err := tx.Delete(&model.UserSubscribeChannel{}, &model.UserSubscribeChannel{UserID: uid, ChannelID: channelID}).Error; err != nil {
					return err
				}
				if current[uid] == model.ChannelSubscribeLevelMarkAndNotify {
					off = append(off, uid)
				}

			case model.ChannelSubscribeLevelMark:
				if _, ok := current[uid]; ok {
					if err := tx.Model(model.UserSubscribeChannel{}).
						Where(&model.UserSubscribeChannel{UserID: uid, ChannelID: channelID}).
						Updates(map[string]interface{}{"mark": true, "notify": false}).
						Error; err != nil {
						return err
					}
				} else {
					if err := tx.Create(&model.UserSubscribeChannel{UserID: uid, ChannelID: channelID, Mark: true, Notify: false}).Error; err != nil {
						if gormutil.IsMySQLForeignKeyConstraintFailsError(err) {
							continue // 存在しないユーザーは無視
						}
						return err
					}
				}
				if current[uid] == model.ChannelSubscribeLevelMarkAndNotify {
					off = append(off, uid)
				}

			case model.ChannelSubscribeLevelMarkAndNotify:
				if _, ok := current[uid]; ok {
					if err := tx.Model(model.UserSubscribeChannel{}).
						Where(&model.UserSubscribeChannel{UserID: uid, ChannelID: channelID}).
						Updates(map[string]interface{}{"mark": true, "notify": true}).
						Error; err != nil {
						return err
					}
				} else {
					if err := tx.Create(&model.UserSubscribeChannel{UserID: uid, ChannelID: channelID, Mark: true, Notify: true}).Error; err != nil {
						if gormutil.IsMySQLForeignKeyConstraintFailsError(err) {
							continue // 存在しないユーザーは無視
						}
						return err
					}
				}
				on = append(on, uid)
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	if len(on) > 0 || len(off) > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.ChannelSubscribersChanged,
			Fields: hub.Fields{
				"channel_id":     channelID,
				"subscriber_ids": append(on, off...),
			},
		})
	}
	return on, off, nil
}

// GetChannelSubscriptions implements ChannelRepository interface.
func (repo *Repository) GetChannelSubscriptions(query repository.ChannelSubscriptionQuery) ([]*model.UserSubscribeChannel, error) {
	tx := repo.db

	if query.UserID.Valid {
		tx = tx.Where("user_id = ?", query.UserID.V)
	}
	if query.ChannelID.Valid {
		tx = tx.Where("channel_id = ?", query.ChannelID.V)
	}
	switch query.Level {
	case model.ChannelSubscribeLevelMark:
		tx = tx.Where("mark = true AND notify = false")
	case model.ChannelSubscribeLevelMarkAndNotify:
		tx = tx.Where("mark = true AND notify = true")
	default:
		tx = tx.Where("mark = true OR notify = true")
	}

	result := make([]*model.UserSubscribeChannel, 0)
	err := tx.Find(&result).Error
	return result, err
}

// GetChannelEvents implements ChannelRepository interface.
func (repo *Repository) GetChannelEvents(query repository.ChannelEventsQuery) (events []*model.ChannelEvent, more bool, err error) {
	events = make([]*model.ChannelEvent, 0)

	tx := repo.db
	if query.Asc {
		tx = tx.Order("date_time")
	} else {
		tx = tx.Order("date_time DESC")
	}

	if query.Channel != uuid.Nil {
		tx = tx.Where("channel_id = ?", query.Channel)
	}

	if query.Inclusive {
		if query.Since.Valid {
			tx = tx.Where("date_time >= ?", query.Since.V)
		}
		if query.Until.Valid {
			tx = tx.Where("date_time <= ?", query.Until.V)
		}
	} else {
		if query.Since.Valid {
			tx = tx.Where("date_time > ?", query.Since.V)
		}
		if query.Until.Valid {
			tx = tx.Where("date_time < ?", query.Until.V)
		}
	}

	if query.Offset > 0 {
		tx = tx.Offset(query.Offset)
	}

	if query.Limit > 0 {
		err = tx.Limit(query.Limit + 1).Find(&events).Error
		if len(events) > query.Limit {
			return events[:len(events)-1], true, err
		}
	} else {
		err = tx.Find(&events).Error
	}
	return events, false, err
}

// RecordChannelEvent implements ChannelRepository interface.
func (repo *Repository) RecordChannelEvent(channelID uuid.UUID, eventType model.ChannelEventType, detail model.ChannelEventDetail, datetime time.Time) error {
	return repo.db.Create(&model.ChannelEvent{
		EventID:   uuid.Must(uuid.NewV7()),
		ChannelID: channelID,
		EventType: eventType,
		Detail:    detail,
		DateTime:  datetime,
	}).Error
}

// GetChannelStats implements ChannelRepository interface.
func (repo *Repository) GetChannelStats(channelID uuid.UUID, excludeDeletedMessages bool) (*repository.ChannelStats, error) {
	if channelID == uuid.Nil {
		return nil, repository.ErrNilID
	}

	if ok, err := gormutil.RecordExists(repo.db, &model.Channel{ID: channelID}); err != nil {
		return nil, err
	} else if !ok {
		return nil, repository.ErrNotFound
	}

	var stats repository.ChannelStats
	query := repo.db.Unscoped().
		Model(&model.Message{}).
		Select("COUNT(channel_id) AS total_message_count").
		Where(&model.Message{ChannelID: channelID})

	if excludeDeletedMessages {
		query = query.Where("deleted_at IS NULL")
	}

	if err := query.Find(&stats.TotalMessageCount).Error; err != nil {
		return nil, err
	}

	query = repo.db.Unscoped().
		Model(&model.Message{}).
		Select("stamp_id AS id", "COUNT(stamp_id) AS count", "SUM(count) As total").
		Joins("JOIN messages_stamps ON id = messages_stamps.message_id").
		Where(&model.Message{ChannelID: channelID})

	if excludeDeletedMessages {
		query = query.Where("deleted_at IS NULL")
	}

	if err := query.
		Group("stamp_id").
		Order("count DESC").
		Find(&stats.Stamps).
		Error; err != nil {
		return nil, err
	}

	query = repo.db.Unscoped().
		Model(&model.Message{}).
		Select("user_id AS id", "COUNT(user_id) AS message_count").
		Where(&model.Message{ChannelID: channelID})

	if excludeDeletedMessages {
		query = query.Where("deleted_at IS NULL")
	}

	if err := query.
		Group("user_id").
		Order("message_count DESC").
		Find(&stats.Users).
		Error; err != nil {
		return nil, err
	}

	stats.DateTime = time.Now()
	return &stats, nil
}
