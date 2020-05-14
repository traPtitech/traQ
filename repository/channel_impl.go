package repository

import (
	"bytes"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/validator"
	"go.uber.org/zap"
	"time"
)

var (
	dmChannelRootUUID  = uuid.Must(uuid.FromString(model.DirectMessageChannelRootID))
	pubChannelRootUUID = uuid.Nil
)

// CreatePublicChannel implements ChannelRepository interface.
func (repo *GormRepository) CreatePublicChannel(name string, parent, creatorID uuid.UUID) (*model.Channel, error) {
	// チャンネル名検証
	if !validator.ChannelRegex.MatchString(name) {
		return nil, ArgError("name", "invalid name")
	}

	repo.chTree.mu.Lock()
	defer repo.chTree.mu.Unlock()

	if repo.chTree.isChildPresent(name, parent) {
		return nil, ErrAlreadyExists
	}

	switch parent {
	case pubChannelRootUUID: // ルート
		break
	case dmChannelRootUUID: // DMルート
		return nil, ErrForbidden
	default: // ルート以外
		// 親チャンネル検証
		if !repo.chTree.isChannelPresent(parent) {
			return nil, ErrNotFound
		}
		// 深さを検証
		if len(repo.chTree.getAscendantIDs(parent))+2 > model.MaxChannelDepth {
			return nil, ErrChannelDepthLimitation
		}
	}

	ch := &model.Channel{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		ParentID:  parent,
		CreatorID: creatorID,
		UpdaterID: creatorID,
		IsPublic:  true,
		IsForced:  false,
		IsVisible: true,
	}
	if err := repo.db.Create(ch).Error; err != nil {
		return nil, err
	}
	repo.chTree.add(ch)
	repo.hub.Publish(hub.Message{
		Name: event.ChannelCreated,
		Fields: hub.Fields{
			"channel_id": ch.ID,
			"channel":    ch,
			"private":    false,
		},
	})

	if ch.ParentID != uuid.Nil {
		// ロギング
		go repo.recordChannelEvent(ch.ParentID, model.ChannelEventChildCreated, model.ChannelEventDetail{
			"userId":    ch.CreatorID,
			"channelId": ch.ID,
		}, ch.UpdatedAt)
	}
	return ch, nil
}

// UpdateChannel implements ChannelRepository interface.
func (repo *GormRepository) UpdateChannel(channelID uuid.UUID, args UpdateChannelArgs) error {
	if channelID == uuid.Nil {
		return ErrNilID
	}

	repo.chTree.mu.Lock()
	defer repo.chTree.mu.Unlock()

	var (
		ch                model.Channel
		nameChanged       bool
		topicChanged      bool
		visibilityChanged bool
		forcedChanged     bool
		parentChanged     bool
		nameBefore        string
		parentBefore      uuid.UUID
		topicBefore       string
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&ch, &model.Channel{ID: channelID}).Error; err != nil {
			return convertError(err)
		}

		data := map[string]interface{}{"updater_id": args.UpdaterID}
		if args.Topic.Valid && ch.Topic != args.Topic.String {
			topicBefore = ch.Topic
			topicChanged = true
			data["topic"] = args.Topic.String
		}
		if args.Visibility.Valid && ch.IsVisible != args.Visibility.Bool {
			visibilityChanged = true
			data["is_visible"] = args.Visibility.Bool
		}
		if args.ForcedNotification.Valid && ch.IsForced != args.ForcedNotification.Bool {
			forcedChanged = true
			data["is_forced"] = args.ForcedNotification.Bool
		}
		if args.Name.Valid || args.Parent.Valid {
			// ダイレクトメッセージチャンネルの名前・親は変更不可能
			if ch.IsDMChannel() {
				return ErrForbidden
			}

			// チャンネル名重複を確認
			{
				var (
					n string
					p uuid.UUID
				)

				if args.Name.Valid {
					n = args.Name.String
				} else {
					n = ch.Name
				}
				if args.Parent.Valid {
					p = args.Parent.UUID
				} else {
					p = ch.ParentID
				}

				if repo.chTree.isChildPresent(n, p) {
					return ErrAlreadyExists
				}
			}

			if args.Name.Valid {
				// チャンネル名検証
				if !validator.ChannelRegex.MatchString(args.Name.String) {
					return ArgError("args.Name", "invalid name")
				}

				nameBefore = ch.Name
				nameChanged = true
				data["name"] = args.Name.String
			}
			if args.Parent.Valid {
				// チャンネル階層検証
				switch args.Parent.UUID {
				case pubChannelRootUUID:
					// ルートチャンネル
					break
				case dmChannelRootUUID:
					// DMチャンネルには出来ない
					return ArgError("args.Parent", "invalid parent channel")
				default:
					// 親チャンネル検証
					if !repo.chTree.isChannelPresent(args.Parent.UUID) {
						return ArgError("args.Parent", "invalid parent channel")
					}

					// 深さを検証
					ascs := append(repo.chTree.getAscendantIDs(args.Parent.UUID), args.Parent.UUID)
					for _, id := range ascs {
						if id == ch.ID {
							return ErrChannelDepthLimitation // ループ検出
						}
					}
					if len(ascs)+1+repo.chTree.getChannelDepth(ch.ID) > model.MaxChannelDepth {
						return ErrChannelDepthLimitation
					}
				}

				parentBefore = ch.ParentID
				parentChanged = true
				data["parent_id"] = args.Parent.UUID
			}
		}

		return tx.Model(&ch).Updates(data).Error
	})
	if err != nil {
		return err
	}

	if nameChanged || parentChanged {
		repo.chTree.move(channelID, args.Parent, args.Name) // ツリー更新
	}

	if forcedChanged || visibilityChanged || topicChanged || nameChanged || parentChanged {
		repo.hub.Publish(hub.Message{
			Name: event.ChannelUpdated,
			Fields: hub.Fields{
				"channel_id": channelID,
				"private":    !ch.IsPublic,
			},
		})
		archived := optional.NewBool(!args.Visibility.Bool, args.Visibility.Valid)
		repo.chTree.update(channelID, args.Topic, archived, args.ForcedNotification)
		if topicChanged {
			repo.hub.Publish(hub.Message{
				Name: event.ChannelTopicUpdated,
				Fields: hub.Fields{
					"channel_id": channelID,
					"topic":      args.Topic.String,
					"updater_id": args.UpdaterID,
				},
			})

			go repo.recordChannelEvent(channelID, model.ChannelEventTopicChanged, model.ChannelEventDetail{
				"userId": args.UpdaterID,
				"before": topicBefore,
				"after":  args.Topic.String,
			}, ch.UpdatedAt)
		}
		if forcedChanged {
			go repo.recordChannelEvent(channelID, model.ChannelEventForcedNotificationChanged, model.ChannelEventDetail{
				"userId": args.UpdaterID,
				"force":  args.ForcedNotification.Bool,
			}, ch.UpdatedAt)
		}
		if visibilityChanged {
			go repo.recordChannelEvent(channelID, model.ChannelEventVisibilityChanged, model.ChannelEventDetail{
				"userId":     args.UpdaterID,
				"visibility": args.Visibility.Bool,
			}, ch.UpdatedAt)
		}
		if nameChanged {
			go repo.recordChannelEvent(channelID, model.ChannelEventNameChanged, model.ChannelEventDetail{
				"userId": args.UpdaterID,
				"before": nameBefore,
				"after":  args.Name.String,
			}, ch.UpdatedAt)
		}
		if parentChanged {
			go repo.recordChannelEvent(channelID, model.ChannelEventParentChanged, model.ChannelEventDetail{
				"userId": args.UpdaterID,
				"before": parentBefore,
				"after":  args.Parent.UUID,
			}, ch.UpdatedAt)
		}
	}
	return nil
}

// GetChannel implements ChannelRepository interface.
func (repo *GormRepository) GetChannel(channelID uuid.UUID) (*model.Channel, error) {
	if channelID == uuid.Nil {
		return nil, ErrNotFound
	}
	var ch model.Channel
	if err := repo.db.First(&ch, &model.Channel{ID: channelID}).Error; err != nil {
		return nil, convertError(err)
	}
	return &ch, nil
}

// GetChannelByMessageID implements ChannelRepository interface.
func (repo *GormRepository) GetChannelByMessageID(messageID uuid.UUID) (*model.Channel, error) {
	if messageID == uuid.Nil {
		return nil, ErrNotFound
	}

	channel := &model.Channel{}
	err := repo.db.
		Where("id = ?", repo.db.
			Model(&model.Message{}).
			Select("messages.channel_id").
			Where(&model.Message{ID: messageID}).
			SubQuery()).
		Take(channel).
		Error
	if err != nil {
		return nil, convertError(err)
	}
	return channel, nil
}

// GetChannelsByUserID implements ChannelRepository interface.
func (repo *GormRepository) GetChannelsByUserID(userID uuid.UUID) (channels []*model.Channel, err error) {
	channels = make([]*model.Channel, 0)
	if userID == uuid.Nil {
		err = repo.db.Where(&model.Channel{IsPublic: true}).Find(&channels).Error
		return channels, err
	}
	err = repo.db.
		Joins("LEFT JOIN users_private_channels ON users_private_channels.channel_id = channels.id").
		Where("channels.is_public = true OR users_private_channels.user_id = ?", userID).
		Find(&channels).
		Error
	return channels, err
}

// GetDirectMessageChannel implements ChannelRepository interface.
func (repo *GormRepository) GetDirectMessageChannel(user1, user2 uuid.UUID) (*model.Channel, error) {
	if user1 == uuid.Nil || user2 == uuid.Nil {
		return nil, ErrNilID
	}

	// user1 <= user2 になるように入れかえ
	if bytes.Compare(user1.Bytes(), user2.Bytes()) == 1 {
		t := user1
		user1 = user2
		user2 = t
	}

	// チャンネル存在確認
	var channel model.Channel
	err := repo.db.
		Where("id = (SELECT channel_id FROM dm_channel_mappings WHERE user1 = ? AND user2 = ?)", user1, user2).
		First(&channel).
		Error
	if err == nil {
		return &channel, nil
	} else if !gorm.IsRecordNotFoundError(err) {
		return nil, err
	}

	// 存在しなかったので作成
	channel = model.Channel{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      "dm_" + random.AlphaNumeric(17),
		ParentID:  dmChannelRootUUID,
		IsPublic:  false,
		IsVisible: true,
		IsForced:  false,
	}

	arr := []interface{}{
		&channel,
		&model.DMChannelMapping{ChannelID: channel.ID, User1: user1, User2: user2},
		&model.UsersPrivateChannel{UserID: user1, ChannelID: channel.ID},
	}
	if user1 != user2 {
		arr = append(arr, &model.UsersPrivateChannel{UserID: user2, ChannelID: channel.ID})
	}

	err = repo.db.Transaction(func(tx *gorm.DB) error {
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
			"channel_id": channel.ID,
			"channel":    &channel,
			"private":    true,
		},
	})
	return &channel, nil
}

// GetDirectMessageChannelMapping implements ChannelRepository interface.
func (repo *GormRepository) GetDirectMessageChannelMapping(userID uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	if userID == uuid.Nil {
		return map[uuid.UUID]uuid.UUID{}, nil
	}

	var mappings []model.DMChannelMapping
	if err := repo.db.Where("user1 = ? OR user2 = ?", userID, userID).Find(&mappings).Error; err != nil {
		return nil, err
	}

	result := map[uuid.UUID]uuid.UUID{}
	for _, ch := range mappings {
		if ch.User1 != userID {
			result[ch.ChannelID] = ch.User1
		} else {
			result[ch.ChannelID] = ch.User2
		}
	}
	return result, nil
}

// IsChannelAccessibleToUser implements ChannelRepository interface.
func (repo *GormRepository) IsChannelAccessibleToUser(userID, channelID uuid.UUID) (bool, error) {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return false, nil
	}
	c := 0
	err := repo.db.
		Model(&model.Channel{}).
		Joins("LEFT JOIN users_private_channels ON users_private_channels.channel_id = channels.id").
		Where("(channels.is_public = true OR users_private_channels.user_id = ?) AND channels.id = ? AND channels.deleted_at IS NULL", userID, channelID).
		Limit(1).
		Count(&c).
		Error
	return c > 0, err
}

// GetChildrenChannelIDs implements ChannelRepository interface.
func (repo *GormRepository) GetChildrenChannelIDs(channelID uuid.UUID) (children []uuid.UUID, err error) {
	return repo.chTree.GetChildrenIDs(channelID), nil
}

// GetPrivateChannelMemberIDs implements ChannelRepository interface.
func (repo *GormRepository) GetPrivateChannelMemberIDs(channelID uuid.UUID) (users []uuid.UUID, err error) {
	users = make([]uuid.UUID, 0)
	if channelID == uuid.Nil {
		return users, nil
	}
	err = repo.db.
		Model(&model.UsersPrivateChannel{}).
		Where(&model.UsersPrivateChannel{ChannelID: channelID}).
		Pluck("user_id", &users).
		Error
	return users, err
}

// ChangeChannelSubscription implements ChannelRepository interface.
func (repo *GormRepository) ChangeChannelSubscription(channelID uuid.UUID, args ChangeChannelSubscriptionArgs) error {
	if channelID == uuid.Nil {
		return ErrNilID
	}

	var (
		on  = make([]uuid.UUID, 0)
		off = make([]uuid.UUID, 0)
	)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		// 現在のチャンネルの購読設定を全取得
		var _current []*model.UserSubscribeChannel
		if err := tx.Where(&model.UserSubscribeChannel{ChannelID: channelID}).Find(&_current).Error; err != nil {
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

				if err := tx.Delete(&model.UserSubscribeChannel{UserID: uid, ChannelID: channelID}).Error; err != nil {
					return err
				}
				off = append(off, uid)

			case model.ChannelSubscribeLevelMark:
				if _, ok := current[uid]; ok {
					if err := tx.Model(model.UserSubscribeChannel{}).Where(&model.UserSubscribeChannel{UserID: uid, ChannelID: channelID}).Updates(map[string]bool{"mark": true, "notify": false}).Error; err != nil {
						return err
					}
				} else {
					if err := tx.Create(&model.UserSubscribeChannel{UserID: uid, ChannelID: channelID, Mark: true, Notify: false}).Error; err != nil {
						if isMySQLForeignKeyConstraintFailsError(err) {
							continue // 存在しないユーザーは無視
						}
						return err
					}
				}

			case model.ChannelSubscribeLevelMarkAndNotify:
				if _, ok := current[uid]; ok {
					if err := tx.Model(model.UserSubscribeChannel{}).Where(&model.UserSubscribeChannel{UserID: uid, ChannelID: channelID}).Updates(map[string]bool{"mark": true, "notify": true}).Error; err != nil {
						return err
					}
				} else {
					if err := tx.Create(&model.UserSubscribeChannel{UserID: uid, ChannelID: channelID, Mark: true, Notify: true}).Error; err != nil {
						if isMySQLForeignKeyConstraintFailsError(err) {
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
		return err
	}

	// ロギング
	if len(on) > 0 || len(off) > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.ChannelSubscribersChanged,
			Fields: hub.Fields{
				"channel_id": channelID,
			},
		})
		go repo.recordChannelEvent(channelID, model.ChannelEventSubscribersChanged, model.ChannelEventDetail{
			"userId": args.UpdaterID,
			"on":     on,
			"off":    off,
		}, time.Now())
	}
	return nil
}

// GetChannelSubscriptions implements ChannelRepository interface.
func (repo *GormRepository) GetChannelSubscriptions(query ChannelSubscriptionQuery) ([]*model.UserSubscribeChannel, error) {
	tx := repo.db

	if query.UserID.Valid {
		tx = tx.Where("user_id = ?", query.UserID.UUID)
	}
	if query.ChannelID.Valid {
		tx = tx.Where("channel_id = ?", query.ChannelID.UUID)
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
func (repo *GormRepository) GetChannelEvents(query ChannelEventsQuery) (events []*model.ChannelEvent, more bool, err error) {
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
			tx = tx.Where("date_time >= ?", query.Since.Time)
		}
		if query.Until.Valid {
			tx = tx.Where("date_time <= ?", query.Until.Time)
		}
	} else {
		if query.Since.Valid {
			tx = tx.Where("date_time > ?", query.Since.Time)
		}
		if query.Until.Valid {
			tx = tx.Where("date_time < ?", query.Until.Time)
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
func (repo *GormRepository) RecordChannelEvent(channelID uuid.UUID, eventType model.ChannelEventType, detail model.ChannelEventDetail, datetime time.Time) error {
	return repo.db.Create(&model.ChannelEvent{
		EventID:   uuid.Must(uuid.NewV4()),
		ChannelID: channelID,
		EventType: eventType,
		Detail:    detail,
		DateTime:  datetime,
	}).Error
}

// GetChannelStats implements ChannelRepository interface.
func (repo *GormRepository) GetChannelStats(channelID uuid.UUID) (*ChannelStats, error) {
	if channelID == uuid.Nil {
		return nil, ErrNotFound
	}

	if ok, err := dbExists(repo.db, &model.Channel{ID: channelID}); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrNotFound
	}

	var stats ChannelStats
	stats.DateTime = time.Now()
	return &stats, repo.db.Unscoped().Model(&model.Message{}).Where(&model.Message{ChannelID: channelID}).Count(&stats.TotalMessageCount).Error
}

// GetPublicChannelTree implements ChannelRepository interface.
func (repo *GormRepository) GetPublicChannelTree() ChannelTree {
	return repo.chTree
}

func (repo *GormRepository) recordChannelEvent(channelID uuid.UUID, eventType model.ChannelEventType, detail model.ChannelEventDetail, datetime time.Time) {
	err := repo.RecordChannelEvent(channelID, eventType, detail, datetime)
	if err != nil {
		repo.logger.Warn("Recording channel event failed", zap.Error(err), zap.Stringer("channelID", channelID), zap.Stringer("type", eventType), zap.Any("detail", detail), zap.Time("datetime", datetime))
	}
}
