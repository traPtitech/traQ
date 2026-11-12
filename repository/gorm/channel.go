package gorm

import (
	"bytes"
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/gormutil"
	"github.com/traPtitech/traQ/utils/set"
)

var dmChannelRootUUID = uuid.Must(uuid.FromString(model.DirectMessageChannelRootID))

// CreateChannel implements ChannelRepository interface.
func (repo *Repository) CreateChannel(ctx context.Context, ch model.Channel, privateMembers set.UUID, channelType model.ChannelType) (*model.Channel, error) {
	arr := []interface{}{&ch}

	ch.ID = uuid.Must(uuid.NewV7())
	ch.Type = model.ChannelTypePublic
	ch.DeletedAt = gorm.DeletedAt{}

	if len(privateMembers) > 0 {
		ch.Type = model.ChannelTypeDM
		ch.IsForced = false
		for uid := range privateMembers {
			arr = append(arr, &model.UsersPrivateChannel{
				UserID:    uid,
				ChannelID: ch.ID,
			})
		}
	}

	if channelType == model.ChannelTypeDM {
		ch.ParentID = dmChannelRootUUID
		ch.Type = model.ChannelTypeDM
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

	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, v := range arr {
			if err := tx.Create(v).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if gormutil.IsMySQLDuplicatedRecordErr(err) {
			return nil, repository.ErrAlreadyExists
		}
		return nil, err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ChannelCreated,
		Fields: hub.Fields{
			"channel_id": ch.ID,
			"channel":    &ch,
			"private":    ch.IsDMChannel(),
		},
	})
	return &ch, nil
}

// CreateThread implements ChannelRepository interface.
func (repo *Repository) CreateThread(ctx context.Context, ch model.Channel, th model.Thread) (*model.Thread, error) {
	arr := []interface{}{&ch, &th}

	ch.ID = uuid.Must(uuid.NewV7())
	th.ChannelID = ch.ID
	ch.Type = model.ChannelTypeThread
	ch.DeletedAt = gorm.DeletedAt{}

	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, v := range arr {
			if err := tx.Create(v).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		if gormutil.IsMySQLDuplicatedRecordErr(err) {
			return nil, repository.ErrAlreadyExists
		}
		return nil, err
	}

	repo.hub.Publish(hub.Message{
		Name: event.ThreadCreated,
		Fields: hub.Fields{
			"channel_id": ch.ID,
			"thread":     &ch,
		},
	})
	th.Channel = &ch
	return &th, nil
}

// UpdateChannel implements ChannelRepository interface.
func (repo *Repository) UpdateChannel(ctx context.Context, channelID uuid.UUID, args repository.UpdateChannelArgs) (*model.Channel, error) {
	if channelID == uuid.Nil {
		return nil, repository.ErrNilID
	}

	var ch model.Channel
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		if gormutil.IsMySQLDuplicatedRecordErr(err) {
			return nil, repository.ErrAlreadyExists
		}
		return nil, err
	}

	repo.hub.Publish(hub.Message{
		Name: event.ChannelUpdated,
		Fields: hub.Fields{
			"channel_id": channelID,
			"private":    ch.IsDMChannel(),
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

// UpdateThread implements ChannelRepository interface.
func (repo *Repository) UpdateThread(ctx context.Context, channelID uuid.UUID, args repository.UpdateThreadArgs) (*model.Channel, error) {
	if channelID == uuid.Nil {
		return nil, repository.ErrNilID
	}

	var ch model.Channel
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&ch, &model.Channel{ID: channelID}).Error; err != nil {
			return convertError(err)
		}

		data := map[string]interface{}{"updater_id": args.UpdaterID}
		if args.Visibility.Valid {
			data["is_visible"] = args.Visibility.V
		}
		if args.Name.Valid {
			data["name"] = args.Name.V
		}

		if err := tx.Model(&ch).Updates(data).Error; err != nil {
			return err
		}
		return tx.First(&ch, &model.Channel{ID: channelID}).Error
	})
	if err != nil {
		if gormutil.IsMySQLDuplicatedRecordErr(err) {
			return nil, repository.ErrAlreadyExists
		}
		return nil, err
	}

	repo.hub.Publish(hub.Message{
		Name: event.ThreadUpdated,
		Fields: hub.Fields{
			"channel_id": channelID,
		},
	})
	return &ch, nil
}

// ArchiveChannels implements ChannelRepository interface.
func (repo *Repository) ArchiveChannels(ctx context.Context, ids []uuid.UUID) ([]*model.Channel, error) {
	var changed []*model.Channel
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
				"private":    ch.IsDMChannel(),
			},
		})
	}
	return changed, nil
}

// GetChannel implements ChannelRepository interface.
func (repo *Repository) GetChannel(ctx context.Context, channelID uuid.UUID) (*model.Channel, error) {
	if channelID == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	var ch model.Channel
	if err := repo.db.WithContext(ctx).First(&ch, &model.Channel{ID: channelID}).Error; err != nil {
		return nil, convertError(err)
	}
	return &ch, nil
}

// GetThread implements ChannelRepository interface.
func (repo *Repository) GetThread(ctx context.Context, channelID uuid.UUID) (*model.Thread, error) {
	if channelID == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	var thread model.Thread
	if err := repo.db.WithContext(ctx).First(&thread, &model.Thread{ChannelID: channelID}).Error; err != nil {
		return nil, convertError(err)
	}
	return &thread, nil
}

// GetThreadChannels implements ChannelRepository interface.
func (repo *Repository) GetThreadChannels(ctx context.Context, query repository.GetThreadChannelQuery) ([]*model.Thread, error) {
	threads := make([]*model.Thread, 0)

	tx := repo.db.WithContext(ctx).Model(&model.Thread{}).Joins("Channel").Where("Channel.parent_id = ?", query.ChannelID).Limit(query.Limit).Offset(query.Offset)

	if !query.Archived {
		tx = tx.Where("Channel.is_visible = ?", true)
	}

	switch query.Sort {
	case repository.ThreadChannelSortCreatedAtAsc:
		tx = tx.Order("created_at ASC")
	case repository.ThreadChannelSortCreatedAtDesc:
		tx = tx.Order("created_at DESC")
	}
	err := tx.Find(&threads).Error
	return threads, err
}

// GetPublicChannels implements ChannelRepository interface.
func (repo *Repository) GetPublicChannels(ctx context.Context) (channels []*model.Channel, err error) {
	channels = make([]*model.Channel, 0)
	return channels, repo.db.
		WithContext(ctx).
		Where(&model.Channel{Type: model.ChannelTypePublic}).
		Find(&channels).
		Error
}

// GetDirectMessageChannel implements ChannelRepository interface.
func (repo *Repository) GetDirectMessageChannel(ctx context.Context, user1, user2 uuid.UUID) (*model.Channel, error) {
	// user1 <= user2 になるように入れかえ
	if bytes.Compare(user1.Bytes(), user2.Bytes()) == 1 {
		t := user1
		user1 = user2
		user2 = t
	}

	// チャンネル存在確認
	var ch model.Channel
	err := repo.db.
		WithContext(ctx).
		Where("id = (SELECT channel_id FROM dm_channel_mappings WHERE user1 = ? AND user2 = ?)", user1, user2).
		First(&ch).
		Error
	if err != nil {
		return nil, convertError(err)
	}
	return &ch, nil
}

// GetDirectMessageChannelMapping implements ChannelRepository interface.
func (repo *Repository) GetDirectMessageChannelMapping(ctx context.Context, userID uuid.UUID) (mappings []*model.DMChannelMapping, err error) {
	mappings = make([]*model.DMChannelMapping, 0)
	if userID == uuid.Nil {
		return
	}
	return mappings, repo.db.
		WithContext(ctx).
		Where("user1 = ? OR user2 = ?", userID, userID).
		Find(&mappings).
		Error
}

// GetPrivateChannelMemberIDs implements ChannelRepository interface.
func (repo *Repository) GetPrivateChannelMemberIDs(ctx context.Context, channelID uuid.UUID) (users []uuid.UUID, err error) {
	users = make([]uuid.UUID, 0)
	if channelID == uuid.Nil {
		return users, nil
	}
	return users, repo.db.
		WithContext(ctx).
		Model(&model.UsersPrivateChannel{}).
		Where(&model.UsersPrivateChannel{ChannelID: channelID}).
		Pluck("user_id", &users).
		Error
}

// ChangeChannelSubscription implements ChannelRepository interface.
func (repo *Repository) ChangeChannelSubscription(ctx context.Context, channelID uuid.UUID, args repository.ChangeChannelSubscriptionArgs) (on []uuid.UUID, off []uuid.UUID, err error) {
	if channelID == uuid.Nil {
		return nil, nil, repository.ErrNilID
	}

	on = make([]uuid.UUID, 0)
	off = make([]uuid.UUID, 0)

	uids := make([]uuid.UUID, 0)
	for uid := range args.Subscription {
		uids = append(uids, uid)
	}

	err = repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Where(&model.UserSubscribeChannel{ChannelID: channelID})
		if !args.Replace {
			query = query.Where("user_id IN (?)", uids)
		}
		var _current []*model.UserSubscribeChannel
		if err := query.Find(&_current).Error; err != nil {
			return err
		}
		current := make(map[uuid.UUID]model.ChannelSubscribeLevel, len(_current))
		for _, s := range _current {
			current[s.UserID] = s.GetLevel()
		}

		subscriptions := make(map[uuid.UUID]model.ChannelSubscribeLevel, len(args.Subscription))
		if args.Replace {
			for uid, level := range current {
				if level == model.ChannelSubscribeLevelMarkAndNotify {
					subscriptions[uid] = model.ChannelSubscribeLevelNone
				}
			}
		}
		for uid, level := range args.Subscription {
			subscriptions[uid] = level
		}

		for uid, level := range subscriptions {
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
func (repo *Repository) GetChannelSubscriptions(ctx context.Context, query repository.ChannelSubscriptionQuery) ([]*model.UserSubscribeChannel, error) {
	tx := repo.db.WithContext(ctx)

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

// GetThreadSubscriptionDefaults implements ChannelRepository interface.
func (repo *Repository) GetThreadSubscriptionDefaults(ctx context.Context, userID uuid.UUID) ([]*model.UserThreadSubscriptionDefault, error) {
	if userID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	result := make([]*model.UserThreadSubscriptionDefault, 0)
	err := repo.db.WithContext(ctx).Where("user_id = ?", userID).Find(&result).Error
	return result, err
}

// SetThreadSubscriptionDefault implements ChannelRepository interface.
func (repo *Repository) SetThreadSubscriptionDefault(ctx context.Context, userID, channelID uuid.UUID, level int) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	if level < -1 || level > 2 {
		return repository.ArgError("level", "level must be between -1 and 2")
	}
	setting := model.UserThreadSubscriptionDefault{
		UserID:    userID,
		ChannelID: channelID,
		Level:     model.ChannelSubscribeLevel(level),
	}

	var changed bool
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var levels []int
		if err := tx.Model(&setting).Where("user_id = ? AND channel_id = ?", userID, channelID).
			Pluck("level", &levels).Error; err != nil {
			return err
		}
		previous := -1
		if len(levels) > 0 {
			previous = levels[0]
		}
		changed = subscriptionClass(previous) != subscriptionClass(level)

		if level == -1 {
			return tx.Delete(&setting).Error
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "channel_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"level"}),
		}).Create(&setting).Error
	})
	if err != nil {
		return err
	}

	if changed {
		repo.hub.Publish(hub.Message{
			Name:   event.ThreadDefaultSubscribersChanged,
			Fields: hub.Fields{"channel_id": channelID, "subscriber_ids": []uuid.UUID{userID}},
		})
	}
	return nil
}

// GetThreadSubscriptions implements ChannelRepository interface.
func (repo *Repository) GetThreadSubscriptions(ctx context.Context, userID uuid.UUID) ([]*model.UserThreadSubscription, error) {
	if userID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	result := make([]*model.UserThreadSubscription, 0)
	err := repo.db.WithContext(ctx).Where("user_id = ?", userID).Find(&result).Error
	return result, err
}

// GetThreadSubscription implements ChannelRepository interface.
func (repo *Repository) GetThreadSubscription(ctx context.Context, userID, channelID uuid.UUID) (*model.UserThreadSubscription, error) {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	var result model.UserThreadSubscription
	if err := repo.db.WithContext(ctx).Where("user_id = ? AND channel_id = ?", userID, channelID).First(&result).Error; err != nil {
		return nil, convertError(err)
	}
	return &result, nil
}

// SetThreadSubscription implements ChannelRepository interface.
func (repo *Repository) SetThreadSubscription(ctx context.Context, userID, channelID uuid.UUID, level int) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	if level < -1 || level > 2 {
		return repository.ArgError("level", "level must be between -1 and 2")
	}
	setting := model.UserThreadSubscription{
		UserID:    userID,
		ChannelID: channelID,
		Level:     model.ChannelSubscribeLevel(level),
	}

	var changed bool
	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var levels []int
		if err := tx.Model(&setting).Where("user_id = ? AND channel_id = ?", userID, channelID).
			Pluck("level", &levels).Error; err != nil {
			return err
		}
		previous := -1
		if len(levels) > 0 {
			previous = levels[0]
		}
		changed = subscriptionClass(previous) != subscriptionClass(level)

		if level == -1 {
			return tx.Delete(&setting).Error
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "channel_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"level"}),
		}).Create(&setting).Error
	})
	if err != nil {
		return err
	}

	if changed {
		repo.hub.Publish(hub.Message{
			Name:   event.ThreadSubscribersChanged,
			Fields: hub.Fields{"channel_id": channelID, "subscriber_ids": []uuid.UUID{userID}},
		})
	}
	return nil
}

func subscriptionClass(level int) int {
	if level == 1 {
		return 0
	}
	return level
}

// GetThreadSubscribers implements ChannelRepository interface.
func (repo *Repository) GetThreadSubscribers(ctx context.Context, threadID uuid.UUID) (users []uuid.UUID, err error) {
	users = make([]uuid.UUID, 0)
	if threadID == uuid.Nil {
		return nil, repository.ErrNilID
	}

	err = repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		users, err = getThreadSubscribers(ctx, tx, threadID)
		return err
	})
	return users, err
}

func getThreadSubscribers(ctx context.Context, tx *gorm.DB, threadID uuid.UUID) ([]uuid.UUID, error) {
	tx = tx.WithContext(ctx)
	users := make([]uuid.UUID, 0)
	var thread model.Channel
	if err := tx.Where("id = ? AND type = ?", threadID, model.ChannelTypeThread).
		First(&thread).Error; err != nil {
		return nil, convertError(err)
	}
	var parent model.Channel
	if err := tx.First(&parent, "id = ?", thread.ParentID).Error; err != nil {
		return nil, convertError(err)
	}

	nonInheritingThreadUsers := tx.Model(&model.UserThreadSubscription{}).
		Select("user_id").Where("channel_id = ? AND level <> -1", threadID)
	nonInheritingDefaultUsers := tx.Model(&model.UserThreadSubscriptionDefault{}).
		Select("user_id").Where("channel_id = ? AND level <> -1", parent.ID)
	defaultNotifyUsers := tx.Model(&model.UserThreadSubscriptionDefault{}).
		Select("user_id").Where("channel_id = ? AND level = ?", parent.ID, model.ChannelSubscribeLevelMarkAndNotify)
	parentNotifyUsers := tx.Model(&model.UserSubscribeChannel{}).
		Select("user_id").Where("channel_id = ? AND notify = ?", parent.ID, true)

	// 1. スレッドの購読レベルが2。
	var threadSubscribers []uuid.UUID
	if err := tx.Model(&model.UserThreadSubscription{}).
		Where("channel_id = ? AND level = ?", threadID, model.ChannelSubscribeLevelMarkAndNotify).
		Pluck("user_id", &threadSubscribers).Error; err != nil {
		return nil, err
	}
	users = append(users, threadSubscribers...)

	// 2. スレッドの購読レベルが継承（-1）で、親チャンネルに対するスレッドのデフォルト購読レベルが2。
	var defaultSubscribers []uuid.UUID
	if err := tx.Model(&model.User{}).
		Where("id NOT IN (?)", nonInheritingThreadUsers).
		Where("id IN (?)", defaultNotifyUsers).
		Pluck("id", &defaultSubscribers).Error; err != nil {
		return nil, err
	}
	users = append(users, defaultSubscribers...)

	// スレッドの購読レベルと、親チャンネルのスレッドのデフォルト購読レベルがともに継承（-1）で、親チャンネルが次のいずれかに該当する。
	//  購読レベルが2。または強制通知チャンネル。
	var parentSubscribers []uuid.UUID
	if err := tx.Model(&model.User{}).
		Where("id NOT IN (?)", nonInheritingThreadUsers).
		Where("id NOT IN (?)", nonInheritingDefaultUsers).
		Where("? OR id IN (?)", parent.IsForced, parentNotifyUsers).
		Pluck("id", &parentSubscribers).Error; err != nil {
		return nil, err
	}
	users = append(users, parentSubscribers...)
	return users, nil
}

// ChangeThreadSubscription implements ChannelRepository interface.
func (repo *Repository) ChangeThreadSubscription(ctx context.Context, threadID uuid.UUID, args repository.ChangeChannelSubscriptionArgs) (on []uuid.UUID, off []uuid.UUID, err error) {
	if threadID == uuid.Nil {
		return nil, nil, repository.ErrNilID
	}
	for _, level := range args.Subscription {
		if level != model.ChannelSubscribeLevelNone && level != model.ChannelSubscribeLevelMarkAndNotify {
			return nil, nil, repository.ArgError("level", "level must be 0 or 2")
		}
	}

	on = make([]uuid.UUID, 0)
	off = make([]uuid.UUID, 0)

	changed := make([]uuid.UUID, 0)
	err = repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		currentSubscribers, err := getThreadSubscribers(ctx, tx, threadID)
		if err != nil {
			return err
		}
		current := make(map[uuid.UUID]bool, len(currentSubscribers))

		subscriptions := make(map[uuid.UUID]model.ChannelSubscribeLevel, len(args.Subscription))
		for _, id := range currentSubscribers {
			current[id] = true
			if args.Replace {
				subscriptions[id] = model.ChannelSubscribeLevelNone
			}
		}
		for id, level := range args.Subscription {
			subscriptions[id] = level
		}

		// 存在しないユーザーを除外してから保存する。
		ids := make([]uuid.UUID, 0, len(subscriptions))
		for id := range subscriptions {
			ids = append(ids, id)
		}
		var existingUsers []uuid.UUID
		if err := tx.Model(&model.User{}).Where("id IN (?)", ids).Pluck("id", &existingUsers).Error; err != nil {
			return err
		}

		var currentSettings []model.UserThreadSubscription
		if err := tx.Where("channel_id = ? AND user_id IN (?)", threadID, existingUsers).Find(&currentSettings).Error; err != nil {
			return err
		}
		currentLevels := make(map[uuid.UUID]int, len(currentSettings))
		for _, setting := range currentSettings {
			currentLevels[setting.UserID] = int(setting.Level)
		}
		settings := make([]model.UserThreadSubscription, 0, len(existingUsers))
		for _, id := range existingUsers {
			level := subscriptions[id]
			if level == model.ChannelSubscribeLevelNone && !current[id] {
				continue // 通知されていないユーザーの設定は維持する。
			}

			settings = append(settings, model.UserThreadSubscription{UserID: id, ChannelID: threadID, Level: level})
			previous, ok := currentLevels[id]
			if !ok {
				previous = -1
			}
			if subscriptionClass(previous) != subscriptionClass(int(level)) {
				changed = append(changed, id)
			}
			if level == model.ChannelSubscribeLevelMarkAndNotify && !current[id] {
				on = append(on, id)
			} else if level == model.ChannelSubscribeLevelNone {
				off = append(off, id)
			}
		}
		if len(settings) == 0 {
			return nil
		}

		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "channel_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"level"}),
		}).Create(&settings).Error
	})
	if err != nil {
		return nil, nil, err
	}

	if len(changed) > 0 {
		repo.hub.Publish(hub.Message{
			Name:   event.ThreadSubscribersChanged,
			Fields: hub.Fields{"channel_id": threadID, "subscriber_ids": changed},
		})
	}

	return on, off, nil
}

// GetChannelEvents implements ChannelRepository interface.
func (repo *Repository) GetChannelEvents(ctx context.Context, query repository.ChannelEventsQuery) (events []*model.ChannelEvent, more bool, err error) {
	events = make([]*model.ChannelEvent, 0)

	tx := repo.db.WithContext(ctx)
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
func (repo *Repository) RecordChannelEvent(ctx context.Context, channelID uuid.UUID, eventType model.ChannelEventType, detail model.ChannelEventDetail, datetime time.Time) error {
	return repo.db.WithContext(ctx).Create(&model.ChannelEvent{
		EventID:   uuid.Must(uuid.NewV7()),
		ChannelID: channelID,
		EventType: eventType,
		Detail:    detail,
		DateTime:  datetime,
	}).Error
}

// GetChannelStats implements ChannelRepository interface.
func (repo *Repository) GetChannelStats(ctx context.Context, channelID uuid.UUID, excludeDeletedMessages bool) (*repository.ChannelStats, error) {
	if channelID == uuid.Nil {
		return nil, repository.ErrNilID
	}

	if ok, err := gormutil.RecordExists(repo.db.WithContext(ctx), &model.Channel{ID: channelID}); err != nil {
		return nil, err
	} else if !ok {
		return nil, repository.ErrNotFound
	}

	var stats repository.ChannelStats
	query := repo.db.WithContext(ctx).Unscoped().
		Model(&model.Message{}).
		Select("COUNT(channel_id) AS total_message_count").
		Where(&model.Message{ChannelID: channelID})

	if excludeDeletedMessages {
		query = query.Where("deleted_at IS NULL")
	}

	if err := query.Find(&stats.TotalMessageCount).Error; err != nil {
		return nil, err
	}

	query = repo.db.WithContext(ctx).Unscoped().
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

	query = repo.db.WithContext(ctx).Unscoped().
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

func (repo *Repository) IsChildPresent(ctx context.Context, name string, parent uuid.UUID) (bool, error) {
	if parent == uuid.Nil {
		return false, repository.ArgError("parent", "cannot be nil")
	}
	var count int64
	err := repo.db.WithContext(ctx).
		Model(&model.Channel{}).
		Where(&model.Channel{Name: name, ParentID: parent}).
		Count(&count).
		Error
	return count > 0, err
}
