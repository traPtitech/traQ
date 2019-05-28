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
	"sync"
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
			"private":    false,
		},
	})

	return ch, nil
}

// CreatePrivateChannel implements ChannelRepository interface. TODO トランザクション
func (repo *GormRepository) CreatePrivateChannel(name string, creatorID uuid.UUID, members []uuid.UUID) (*model.Channel, error) {
	validMember := make([]uuid.UUID, 0, len(members))
	for _, v := range members {
		ok, err := repo.UserExists(v)
		if err != nil {
			return nil, err
		}
		if ok {
			validMember = append(validMember, v)
		}
	}
	if err := validator.ValidateVar(validMember, "min=1"); err != nil {
		return nil, err
	}

	// チャンネル名検証
	if !validator.ChannelRegex.MatchString(name) {
		return nil, ArgError("name", "invalid name")
	}
	if has, err := repo.isChannelPresent(repo.db, name, uuid.Nil); err != nil {
		return nil, err
	} else if has {
		return nil, ErrAlreadyExists
	}

	ch := &model.Channel{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		CreatorID: creatorID,
		UpdaterID: creatorID,
		IsPublic:  false,
		IsForced:  false,
		IsVisible: true,
	}

	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Create(ch).Error; err != nil {
			return err
		}

		// メンバーに追加
		for _, v := range validMember {
			if err := tx.Create(&model.UsersPrivateChannel{UserID: v, ChannelID: ch.ID}).Error; err != nil {
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
			"private":    true,
		},
	})

	return ch, nil
}

// CreateChildChannel implements ChannelRepository interface. TODO トランザクション
func (repo *GormRepository) CreateChildChannel(name string, parentID, creatorID uuid.UUID) (*model.Channel, error) {
	// ダイレクトメッセージルートの子チャンネルは作れない
	if parentID == dmChannelRootUUID {
		return nil, ErrForbidden
	}

	// 親チャンネル検証
	pCh, err := repo.GetChannel(parentID)
	if err != nil {
		return nil, err
	}

	// ダイレクトメッセージの子チャンネルは作れない
	if pCh.IsDMChannel() {
		return nil, ErrForbidden
	}

	// チャンネル名検証
	if !validator.ChannelRegex.MatchString(name) {
		return nil, ArgError("name", "invalid name")
	}
	if has, err := repo.isChannelPresent(repo.db, name, pCh.ID); err != nil {
		return nil, err
	} else if has {
		return nil, ErrAlreadyExists
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

	ch := &model.Channel{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		ParentID:  pCh.ID,
		CreatorID: creatorID,
		UpdaterID: creatorID,
		IsForced:  false,
		IsVisible: true,
	}

	if pCh.IsPublic {
		// 公開チャンネル
		ch.IsPublic = true
		if err := repo.db.Create(ch).Error; err != nil {
			return nil, err
		}
		channelsCounter.Inc()
	} else {
		// 非公開チャンネル
		ch.IsPublic = false

		// 親チャンネルとメンバーは同じ
		ids, err := repo.GetPrivateChannelMemberIDs(pCh.ID)
		if err != nil {
			return nil, err
		}

		err = repo.transact(func(tx *gorm.DB) error {
			if err := tx.Create(ch).Error; err != nil {
				return err
			}

			// メンバーに追加
			for _, v := range ids {
				if err := tx.Create(&model.UsersPrivateChannel{UserID: v, ChannelID: ch.ID}).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	repo.hub.Publish(hub.Message{
		Name: event.ChannelCreated,
		Fields: hub.Fields{
			"channel_id": ch.ID,
			"private":    !ch.IsPublic,
		},
	})
	return ch, nil
}

// UpdateChannelAttributes implements ChannelRepository interface.
func (repo *GormRepository) UpdateChannelAttributes(channelID uuid.UUID, visibility, forced *bool) error {
	if channelID == uuid.Nil {
		return ErrNilID
	}

	data := map[string]interface{}{}
	if visibility != nil {
		data["is_visible"] = *visibility
	}
	if forced != nil {
		data["is_forced"] = *forced
	}

	var ch model.Channel
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.First(&ch, &model.Channel{ID: channelID}).Error; err != nil {
			return convertError(err)
		}
		return tx.Model(&ch).Updates(data).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ChannelUpdated,
		Fields: hub.Fields{
			"channel_id": ch.ID,
			"private":    !ch.IsPublic,
		},
	})
	return nil
}

// UpdateChannelTopic implements ChannelRepository interface.
func (repo *GormRepository) UpdateChannelTopic(channelID uuid.UUID, topic string, updaterID uuid.UUID) error {
	if channelID == uuid.Nil {
		return ErrNilID
	}
	var ch model.Channel
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.First(&ch, &model.Channel{ID: channelID}).Error; err != nil {
			return convertError(err)
		}
		return tx.Model(&ch).Updates(map[string]interface{}{
			"topic":      topic,
			"updater_id": updaterID,
		}).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ChannelUpdated,
		Fields: hub.Fields{
			"channel_id": ch.ID,
			"private":    !ch.IsPublic,
		},
	})
	repo.hub.Publish(hub.Message{
		Name: event.ChannelTopicUpdated,
		Fields: hub.Fields{
			"channel_id": ch.ID,
			"topic":      ch.Topic,
			"updater_id": updaterID,
		},
	})
	return nil
}

// ChangeChannelName implements ChannelRepository interface.
func (repo *GormRepository) ChangeChannelName(channelID uuid.UUID, name string) error {
	if channelID == uuid.Nil {
		return ErrNilID
	}

	// チャンネル名検証
	if !validator.ChannelRegex.MatchString(name) {
		return ArgError("name", "invalid name")
	}

	var ch model.Channel
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.First(&ch, &model.Channel{ID: channelID}).Error; err != nil {
			return convertError(err)
		}

		// ダイレクトメッセージチャンネルの名前は変更不可能
		if ch.IsDMChannel() {
			return ErrForbidden
		}

		// チャンネル名重複を確認
		if exists, err := repo.isChannelPresent(tx, name, ch.ParentID); err != nil {
			return err
		} else if exists {
			return ErrAlreadyExists
		}

		return tx.Model(&ch).Update("name", name).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ChannelUpdated,
		Fields: hub.Fields{
			"channel_id": ch.ID,
			"private":    !ch.IsPublic,
		},
	})

	// チャンネルパスキャッシュの削除
	repo.ChannelPathMap.Delete(channelID)
	ids, _ := repo.getDescendantChannelIDs(repo.db, channelID)
	for _, v := range ids {
		repo.ChannelPathMap.Delete(v)
	}

	return nil
}

// ChangeChannelParent implements ChannelRepository interface. TODO トランザクション
func (repo *GormRepository) ChangeChannelParent(channelID uuid.UUID, parent uuid.UUID) error {
	if channelID == uuid.Nil {
		return ErrNilID
	}

	// チャンネル取得
	ch, err := repo.GetChannel(channelID)
	if err != nil {
		return err
	}

	// ダイレクトメッセージチャンネルの親は変更不可能
	if ch.IsDMChannel() {
		return ErrForbidden
	}

	switch parent {
	case uuid.Nil:
		// ルートチャンネル
		break
	case dmChannelRootUUID:
		// DMチャンネルには出来ない
		return ErrForbidden
	default:
		pCh, err := repo.GetChannel(parent)
		if err != nil {
			return err
		}

		// DMチャンネルの子チャンネルには出来ない
		if pCh.IsDMChannel() {
			return ErrForbidden
		}

		// 親と公開状況が一致しているか
		if ch.IsPublic != pCh.IsPublic {
			return ErrForbidden
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

			pCh, err = repo.GetChannel(pCh.ParentID)
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
		bottom, err := repo.getChannelDepth(repo.db, ch.ID) // 子孫 (自分を含む)
		if err != nil {
			return err
		}
		depth += bottom
		if depth > model.MaxChannelDepth {
			return ErrChannelDepthLimitation
		}
	}

	// チャンネル名検証
	if has, err := repo.isChannelPresent(repo.db, ch.Name, parent); err != nil {
		return err
	} else if has {
		return ErrAlreadyExists
	}

	// 更新
	if err := repo.db.Model(&model.Channel{ID: channelID}).Updates(map[string]interface{}{
		"parent_id": parent,
	}).Error; err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ChannelUpdated,
		Fields: hub.Fields{
			"channel_id": ch.ID,
			"private":    !ch.IsPublic,
		},
	})

	// チャンネルパスキャッシュの削除
	repo.ChannelPathMap.Delete(channelID)
	ids, _ := repo.getDescendantChannelIDs(repo.db, channelID)
	for _, v := range ids {
		repo.ChannelPathMap.Delete(v)
	}

	return nil
}

// DeleteChannel implements ChannelRepository interface.
func (repo *GormRepository) DeleteChannel(channelID uuid.UUID) error {
	if channelID == uuid.Nil {
		return ErrNilID
	}

	deleted := make([]*model.Channel, 0)
	err := repo.transact(func(tx *gorm.DB) error {
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
	if channelID == uuid.Nil {
		return nil, ErrNotFound
	}

	ch := &model.Channel{}
	if err := repo.db.First(ch, &model.Channel{ID: channelID}).Error; err != nil {
		return nil, convertError(err)
	}
	return ch, nil
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

	err = repo.transact(func(tx *gorm.DB) error {
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
			"private":    true,
		},
	})
	return &channel, nil
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

// SubscribeChannel implements ChannelRepository interface.
func (repo *GormRepository) SubscribeChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return ErrNilID
	}
	var s model.UserSubscribeChannel
	return repo.db.FirstOrCreate(&s, &model.UserSubscribeChannel{UserID: userID, ChannelID: channelID}).Error
}

// UnsubscribeChannel implements ChannelRepository interface.
func (repo *GormRepository) UnsubscribeChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return nil
	}
	return repo.db.
		Delete(&model.UserSubscribeChannel{UserID: userID, ChannelID: channelID}).
		Error
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
