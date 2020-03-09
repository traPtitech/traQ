package repository

import (
	"bytes"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/validator"
	"go.uber.org/zap"
	"sync"
	"time"
)

var (
	dmChannelRootUUID  = uuid.Must(uuid.FromString(model.DirectMessageChannelRootID))
	pubChannelRootUUID = uuid.Nil
)

type channelImpl struct {
	ChannelPathMap sync.Map
}

// CreatePublicChannel implements ChannelRepository interface. TODO トランザクション
func (repo *GormRepository) CreatePublicChannel(name string, parent, creatorID uuid.UUID) (*model.Channel, error) {
	// チャンネル名検証
	if !validator.ChannelRegex.MatchString(name) {
		return nil, ArgError("name", "invalid name")
	}
	if has, err := repo.isChannelPresent(repo.db, name, parent); err != nil {
		return nil, err
	} else if has {
		return nil, ErrAlreadyExists
	}

	switch parent {
	case pubChannelRootUUID: // ルート
		break
	case dmChannelRootUUID: // DMルート
		return nil, ErrForbidden
	default: // ルート以外
		// 親チャンネル検証
		pCh, err := repo.GetChannel(parent)
		if err != nil {
			return nil, err
		}

		// DMチャンネルの子チャンネルには出来ない
		if pCh.IsDMChannel() {
			return nil, ErrForbidden
		}

		// 親と公開状況が一致しているか
		if !pCh.IsPublic {
			return nil, ErrForbidden
		}

		// 深さを検証
		for parent, depth := pCh, 2; ; { // 祖先
			if parent.ParentID == uuid.Nil {
				// ルート
				break
			}

			parent, err = repo.GetChannel(parent.ParentID)
			if err != nil {
				if err == ErrNotFound {
					break
				}
				return nil, err
			}
			depth++
			if depth > model.MaxChannelDepth {
				return nil, ErrChannelDepthLimitation
			}
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
	channelsCounter.Inc()
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

				if has, err := repo.isChannelPresent(tx, n, p); err != nil {
					return err
				} else if has {
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
				case uuid.Nil:
					// ルートチャンネル
					break
				case dmChannelRootUUID:
					// DMチャンネルには出来ない
					return ArgError("args.Parent", "invalid parent channel")
				default:
					pCh, err := repo.getChannel(tx, args.Parent.UUID)
					if err != nil {
						if err == ErrNotFound {
							return ArgError("args.Parent", "invalid parent channel")
						}
						return err
					}

					// DMチャンネルの子チャンネルには出来ない
					if pCh.IsDMChannel() {
						return ArgError("args.Parent", "invalid parent channel")
					}

					// 親と公開状況が一致しているか
					if ch.IsPublic != pCh.IsPublic {
						return ArgError("args.Parent", "invalid parent channel")
					}

					// 深さを検証
					depth := 1 // ↑で見た親
					for {      // 祖先
						if pCh.ParentID == uuid.Nil {
							// ルート
							break
						}
						if ch.ID == pCh.ID {
							// ループ検出
							return ErrChannelDepthLimitation
						}

						pCh, err = repo.getChannel(tx, pCh.ParentID)
						if err != nil {
							if err == ErrNotFound {
								break
							}
							return err
						}
						depth++
						if depth >= model.MaxChannelDepth {
							return ErrChannelDepthLimitation
						}
					}
					bottom, err := repo.getChannelDepth(tx, ch.ID) // 子孫 (自分を含む)
					if err != nil {
						return err
					}
					depth += bottom
					if depth > model.MaxChannelDepth {
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
		// チャンネルパスキャッシュの削除
		repo.ChannelPathMap.Delete(channelID)
		ids, _ := repo.getDescendantChannelIDs(repo.db, channelID)
		for _, v := range ids {
			repo.ChannelPathMap.Delete(v)
		}
	}

	if forcedChanged || visibilityChanged || topicChanged || nameChanged || parentChanged {
		repo.hub.Publish(hub.Message{
			Name: event.ChannelUpdated,
			Fields: hub.Fields{
				"channel_id": channelID,
				"private":    !ch.IsPublic,
			},
		})
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

// DeleteChannel implements ChannelRepository interface.
func (repo *GormRepository) DeleteChannel(channelID uuid.UUID) error {
	if channelID == uuid.Nil {
		return ErrNilID
	}

	deleted := make([]*model.Channel, 0)
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		if exists, err := dbExists(tx, &model.Channel{ID: channelID}); err != nil {
			return err
		} else if !exists {
			return ErrNotFound
		}

		desc, err := repo.getDescendantChannelIDs(tx, channelID)
		if err != nil {
			return err
		}
		desc = append(desc, channelID)

		for _, v := range desc {
			ch := model.Channel{}
			if err := tx.First(&ch, &model.Channel{ID: v}).Error; err != nil {
				if gorm.IsRecordNotFoundError(err) {
					continue
				}
				return err
			}
			deleted = append(deleted, &ch)
			if err := tx.Delete(ch).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, v := range deleted {
		repo.ChannelPathMap.Delete(v.ID)
		repo.hub.Publish(hub.Message{
			Name: event.ChannelDeleted,
			Fields: hub.Fields{
				"channel_id": v.ID,
				"private":    !v.IsPublic,
			},
		})
	}
	return err
}

// GetChannel implements ChannelRepository interface.
func (repo *GormRepository) GetChannel(channelID uuid.UUID) (*model.Channel, error) {
	return repo.getChannel(repo.db, channelID)
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
		Name:      "dm_" + utils.RandAlphabetAndNumberString(17),
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
	return repo.getChildrenChannelIDs(repo.db, channelID)
}

// GetChannelPath implements ChannelRepository interface.
func (repo *GormRepository) GetChannelPath(id uuid.UUID) (string, error) {
	if id == uuid.Nil {
		return "", ErrNotFound
	}
	v, ok := repo.ChannelPathMap.Load(id)
	if ok {
		return v.(string), nil
	}

	ch := &model.Channel{}
	if err := repo.db.Take(ch, &model.Channel{ID: id}).Error; err != nil {
		return "", convertError(err)
	}

	var path string
	if pid := ch.ParentID; pid != uuid.Nil {
		parentPath, err := repo.GetChannelPath(pid)
		if err != nil && err != ErrNotFound {
			return "", err
		}
		path = parentPath + "/" + ch.Name
	} else {
		path = ch.Name
	}

	repo.ChannelPathMap.Store(id, path)
	return path, nil
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
		for userID, subscribe := range args.Subscription {
			if userID == uuid.Nil {
				continue
			}
			if subscribe {
				err := tx.Create(&model.UserSubscribeChannel{UserID: userID, ChannelID: channelID}).Error
				switch {
				case err == nil:
					// 成功
					on = append(on, userID)
				case isMySQLDuplicatedRecordErr(err):
					// 既に購読中なので無視
					continue
				case isMySQLForeignKeyConstraintFailsError(err):
					// 存在しないユーザーは無視
					continue
				default:
					return err
				}
			} else {
				result := tx.Delete(&model.UserSubscribeChannel{UserID: userID, ChannelID: channelID})
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected > 0 {
					off = append(off, userID)
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// ロギング
	if len(on) > 0 || len(off) > 0 {
		go repo.recordChannelEvent(channelID, model.ChannelEventSubscribersChanged, model.ChannelEventDetail{
			"userId": args.UpdaterID,
			"on":     on,
			"off":    off,
		}, time.Now())
	}

	return nil
}

// GetSubscribingUserIDs implements ChannelRepository interface.
func (repo *GormRepository) GetSubscribingUserIDs(channelID uuid.UUID) (users []uuid.UUID, err error) {
	users = make([]uuid.UUID, 0)
	if channelID == uuid.Nil {
		return users, nil
	}
	err = repo.db.
		Model(&model.UserSubscribeChannel{}).
		Where(&model.UserSubscribeChannel{ChannelID: channelID}).
		Pluck("user_id", &users).
		Error
	return users, err
}

// GetSubscribedChannelIDs implements ChannelRepository interface.
func (repo *GormRepository) GetSubscribedChannelIDs(userID uuid.UUID) (channels []uuid.UUID, err error) {
	channels = make([]uuid.UUID, 0)
	if userID == uuid.Nil {
		return channels, nil
	}
	err = repo.db.
		Model(&model.UserSubscribeChannel{}).
		Where(&model.UserSubscribeChannel{UserID: userID}).
		Pluck("channel_id", &channels).
		Error
	return channels, err
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

// getParentChannel チャンネルを取得する
func (repo *GormRepository) getChannel(tx *gorm.DB, channelID uuid.UUID) (*model.Channel, error) {
	if channelID == uuid.Nil {
		return nil, ErrNotFound
	}

	ch := &model.Channel{}
	if err := tx.First(ch, &model.Channel{ID: channelID}).Error; err != nil {
		return nil, convertError(err)
	}
	return ch, nil
}

// isChannelPresent チャンネル名が同階層に既に存在するか
func (repo *GormRepository) isChannelPresent(tx *gorm.DB, name string, parent uuid.UUID) (bool, error) {
	c := 0
	err := tx.
		Model(&model.Channel{}).
		Where("parent_id = ? AND name = ?", parent, name).
		Limit(1).
		Count(&c).
		Error
	return c > 0, err
}

// getParentChannel 親のチャンネルを取得する
func (repo *GormRepository) getParentChannel(tx *gorm.DB, channelID uuid.UUID) (*model.Channel, error) {
	if channelID == uuid.Nil {
		return nil, ErrNotFound
	}

	var p []uuid.UUID
	err := tx.
		Model(&model.Channel{}).
		Where(&model.Channel{ID: channelID}).
		Pluck("parent_id", &p).
		Error
	if err != nil {
		return nil, err
	}
	if len(p) == 0 {
		return nil, ErrNotFound
	} else if p[0] == uuid.Nil {
		return nil, nil
	}

	ch := &model.Channel{}
	if err := tx.Take(ch, &model.Channel{ID: p[0]}).Error; err != nil {
		return nil, convertError(err)
	}
	return ch, nil
}

// getChannelDepth 指定したチャンネル木の深さを取得する
func (repo *GormRepository) getChannelDepth(tx *gorm.DB, id uuid.UUID) (int, error) {
	children, err := repo.getChildrenChannelIDs(tx, id)
	if err != nil {
		return 0, err
	}
	max := 0
	for _, v := range children {
		d, err := repo.getChannelDepth(tx, v)
		if err != nil {
			return 0, err
		}
		if max < d {
			max = d
		}
	}
	return max + 1, nil
}

// getChildrenChannelIDs 子チャンネルのIDを取得する
func (repo *GormRepository) getChildrenChannelIDs(tx *gorm.DB, channelID uuid.UUID) (children []uuid.UUID, err error) {
	children = make([]uuid.UUID, 0)
	if channelID == uuid.Nil {
		return children, nil
	}
	err = tx.
		Model(&model.Channel{}).
		Where(&model.Channel{ParentID: channelID}).
		Pluck("id", &children).Error
	return children, err
}

// getDescendantChannelIDs 子孫チャンネルのIDを取得する
func (repo *GormRepository) getDescendantChannelIDs(tx *gorm.DB, channelID uuid.UUID) ([]uuid.UUID, error) {
	var descendants []uuid.UUID
	children, err := repo.getChildrenChannelIDs(tx, channelID)
	if err != nil {
		return nil, err
	}
	descendants = append(descendants, children...)
	for _, v := range children {
		sub, err := repo.getDescendantChannelIDs(tx, v)
		if err != nil {
			return nil, err
		}
		descendants = append(descendants, sub...)
	}
	return descendants, nil
}

// getAscendantChannelIDs 祖先チャンネルのIDを取得する
func (repo *GormRepository) getAscendantChannelIDs(tx *gorm.DB, channelID uuid.UUID) ([]uuid.UUID, error) {
	var ascendants []uuid.UUID
	parent, err := repo.getParentChannel(tx, channelID)
	if err != nil {
		if err == ErrNotFound {
			return nil, nil
		}
		return nil, err
	} else if parent == nil {
		return []uuid.UUID{}, nil
	}
	ascendants = append(ascendants, parent.ID)
	sub, err := repo.getAscendantChannelIDs(tx, parent.ID)
	if err != nil {
		return nil, err
	}
	ascendants = append(ascendants, sub...)
	return ascendants, nil
}

func (repo *GormRepository) recordChannelEvent(channelID uuid.UUID, eventType model.ChannelEventType, detail model.ChannelEventDetail, datetime time.Time) {
	err := repo.RecordChannelEvent(channelID, eventType, detail, datetime)
	if err != nil {
		repo.logger.Warn("Recording channel event failed", zap.Error(err), zap.Stringer("channelID", channelID), zap.Stringer("type", eventType), zap.Any("detail", detail), zap.Time("datetime", datetime))
	}
}
