package impl

import (
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
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

// CreatePublicChannel パブリックチャンネルを作成します
func (repo *RepositoryImpl) CreatePublicChannel(name string, parent, creatorID uuid.UUID) (*model.Channel, error) {
	// チャンネル名検証
	if err := validator.ValidateVar(name, "channel,required"); err != nil {
		return nil, err
	}
	if has, err := repo.IsChannelPresent(name, parent); err != nil {
		return nil, err
	} else if has {
		return nil, repository.ErrAlreadyExists
	}

	switch parent {
	case pubChannelRootUUID: // ルート
		break
	case dmChannelRootUUID: // DMルート
		return nil, repository.ErrForbidden
	default: // ルート以外
		pCh, err := repo.GetChannel(parent)
		if err != nil {
			return nil, err
		}

		// DMチャンネルの子チャンネルには出来ない
		if pCh.IsDMChannel() {
			return nil, repository.ErrForbidden
		}

		// 親と公開状況が一致しているか
		if !pCh.IsPublic {
			return nil, repository.ErrForbidden
		}

		// 深さを検証
		for parent, depth := pCh, 2; ; { // 祖先
			if parent.ParentID == uuid.Nil {
				// ルート
				break
			}

			parent, err = repo.GetChannel(parent.ParentID)
			if err != nil {
				if err == repository.ErrNotFound {
					break
				}
				return nil, err
			}
			depth++
			if depth > model.MaxChannelDepth {
				return nil, repository.ErrChannelDepthLimitation
			}
		}
	}

	ch := &model.Channel{
		ID:        uuid.NewV4(),
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
	repo.hub.Publish(hub.Message{
		Name: event.ChannelCreated,
		Fields: hub.Fields{
			"channel_id": ch.ID,
			"private":    false,
		},
	})

	return ch, nil
}

// CreatePrivateChannel プライベートチャンネルを作成します
func (repo *RepositoryImpl) CreatePrivateChannel(name string, creatorID uuid.UUID, members []uuid.UUID) (*model.Channel, error) {
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
	if err := validator.ValidateVar(name, "channel,required"); err != nil {
		return nil, err
	}
	if has, err := repo.IsChannelPresent(name, uuid.Nil); err != nil {
		return nil, err
	} else if has {
		return nil, repository.ErrAlreadyExists
	}

	ch := &model.Channel{
		ID:        uuid.NewV4(),
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

// CreateChildChannel 子チャンネルを作成します TODO トランザクション
func (repo *RepositoryImpl) CreateChildChannel(name string, parentID, creatorID uuid.UUID) (*model.Channel, error) {
	// ダイレクトメッセージルートの子チャンネルは作れない
	if parentID == dmChannelRootUUID {
		return nil, repository.ErrForbidden
	}

	// 親チャンネル検証
	pCh, err := repo.GetChannel(parentID)
	if err != nil {
		return nil, err
	}

	// ダイレクトメッセージの子チャンネルは作れない
	if pCh.IsDMChannel() {
		return nil, repository.ErrForbidden
	}

	// チャンネル名検証
	if err := validator.ValidateVar(name, "channel,required"); err != nil {
		return nil, err
	}
	if has, err := repo.IsChannelPresent(name, pCh.ID); err != nil {
		return nil, err
	} else if has {
		return nil, repository.ErrAlreadyExists
	}

	// 深さを検証
	for parent, depth := pCh, 2; ; { // 祖先
		if parent.ParentID == uuid.Nil {
			// ルート
			break
		}

		parent, err = repo.GetChannel(parent.ParentID)
		if err != nil {
			if err == repository.ErrNotFound {
				break
			}
			return nil, err
		}
		depth++
		if depth > model.MaxChannelDepth {
			return nil, repository.ErrChannelDepthLimitation
		}
	}

	ch := &model.Channel{
		ID:        uuid.NewV4(),
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

// UpdateChannelAttributes チャンネルの属性を変更します
func (repo *RepositoryImpl) UpdateChannelAttributes(channelID uuid.UUID, visibility, forced *bool) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}

	data := map[string]interface{}{}
	if visibility != nil {
		data["is_visible"] = *visibility
	}
	if forced != nil {
		data["is_forced"] = *forced
	}
	var (
		ch model.Channel
		ok bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&model.Channel{ID: channelID}).First(&ch).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return nil
			}
			return err
		}
		ok = true
		return tx.Model(&model.Channel{ID: channelID}).Updates(data).Error
	})
	if err != nil {
		return err
	}
	if ok {
		repo.hub.Publish(hub.Message{
			Name: event.ChannelUpdated,
			Fields: hub.Fields{
				"channel_id": ch.ID,
				"private":    !ch.IsPublic,
			},
		})
	}
	return nil
}

// UpdateChannelTopic チャンネルトピックを更新します
func (repo *RepositoryImpl) UpdateChannelTopic(channelID uuid.UUID, topic string, updaterID uuid.UUID) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}
	var (
		ch model.Channel
		ok bool
	)
	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Where(&model.Channel{ID: channelID}).First(&ch).Error; err != nil {
			if gorm.IsRecordNotFoundError(err) {
				return nil
			}
			return err
		}
		ok = true
		return tx.Model(&model.Channel{ID: channelID}).Updates(map[string]interface{}{
			"topic":      topic,
			"updater_id": updaterID,
		}).Error
	})
	if err != nil {
		return err
	}
	if ok {
		repo.hub.Publish(hub.Message{
			Name: event.ChannelUpdated,
			Fields: hub.Fields{
				"channel_id": ch.ID,
				"private":    !ch.IsPublic,
			},
		})
	}
	return nil
}

// ChangeChannelName チャンネル名を変更します TODO トランザクション
func (repo *RepositoryImpl) ChangeChannelName(channelID uuid.UUID, name string) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}

	// チャンネル名検証
	if err := validator.ValidateVar(name, "channel,required"); err != nil {
		return err
	}

	// チャンネル取得
	ch, err := repo.GetChannel(channelID)
	if err != nil {
		return err
	}

	// ダイレクトメッセージチャンネルの名前は変更不可能
	if ch.IsDMChannel() {
		return repository.ErrForbidden
	}

	// チャンネル名重複を確認
	if has, err := repo.IsChannelPresent(name, ch.ParentID); err != nil {
		return err
	} else if has {
		return repository.ErrAlreadyExists
	}

	// 更新
	if err := repo.db.Model(&model.Channel{ID: channelID}).Updates(map[string]interface{}{
		"name": name,
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
	ids, _ := repo.GetDescendantChannelIDs(channelID)
	for _, v := range ids {
		repo.ChannelPathMap.Delete(v)
	}

	return nil
}

// ChangeChannelParent チャンネルの親を変更します TODO トランザクション
func (repo *RepositoryImpl) ChangeChannelParent(channelID uuid.UUID, parent uuid.UUID) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}

	// チャンネル取得
	ch, err := repo.GetChannel(channelID)
	if err != nil {
		return err
	}

	// ダイレクトメッセージチャンネルの親は変更不可能
	if ch.IsDMChannel() {
		return repository.ErrForbidden
	}

	switch parent {
	case uuid.Nil:
		// ルートチャンネル
		break
	case dmChannelRootUUID:
		// DMチャンネルには出来ない
		return repository.ErrForbidden
	default:
		pCh, err := repo.GetChannel(parent)
		if err != nil {
			return err
		}

		// DMチャンネルの子チャンネルには出来ない
		if pCh.IsDMChannel() {
			return repository.ErrForbidden
		}

		// 親と公開状況が一致しているか
		if ch.IsPublic != pCh.IsPublic {
			return repository.ErrForbidden
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
				return repository.ErrChannelDepthLimitation
			}

			pCh, err = repo.GetChannel(pCh.ParentID)
			if err != nil {
				if err == repository.ErrNotFound {
					break
				}
				return err
			}
			depth++
			if depth >= model.MaxChannelDepth {
				return repository.ErrChannelDepthLimitation
			}
		}
		bottom, err := repo.GetChannelDepth(ch.ID) // 子孫 (自分を含む)
		if err != nil {
			return err
		}
		depth += bottom
		if depth > model.MaxChannelDepth {
			return repository.ErrChannelDepthLimitation
		}
	}

	// チャンネル名検証
	if has, err := repo.IsChannelPresent(ch.Name, parent); err != nil {
		return err
	} else if has {
		return repository.ErrAlreadyExists
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
	ids, _ := repo.GetDescendantChannelIDs(channelID)
	for _, v := range ids {
		repo.ChannelPathMap.Delete(v)
	}

	return nil
}

// DeleteChannel 子孫チャンネルを含めてチャンネルを削除します
func (repo *RepositoryImpl) DeleteChannel(channelID uuid.UUID) error {
	if channelID == uuid.Nil {
		return repository.ErrNilID
	}

	desc, err := repo.GetDescendantChannelIDs(channelID)
	if err != nil {
		return err
	}
	desc = append(desc, channelID)

	deleted := make([]*model.Channel, 0)
	err = repo.transact(func(tx *gorm.DB) error {
		for _, v := range desc {
			ch := model.Channel{}
			if err := tx.Where(&model.Channel{ID: v}).First(&ch).Error; err != nil {
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

// GetChannel チャンネルを取得する
func (repo *RepositoryImpl) GetChannel(channelID uuid.UUID) (*model.Channel, error) {
	if channelID == uuid.Nil {
		return nil, repository.ErrNotFound
	}

	ch := &model.Channel{}
	if err := repo.db.Where(&model.Channel{ID: channelID}).Take(ch).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return ch, nil
}

// GetChannelByMessageID メッセージIDによってチャンネルを取得
func (repo *RepositoryImpl) GetChannelByMessageID(messageID uuid.UUID) (*model.Channel, error) {
	if messageID == uuid.Nil {
		return nil, repository.ErrNotFound
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
		if gorm.IsRecordNotFoundError(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return channel, nil
}

// GetChannelsByUserID ユーザーから見えるチャンネルの一覧を取得する
func (repo *RepositoryImpl) GetChannelsByUserID(userID uuid.UUID) (channels []*model.Channel, err error) {
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

// GetOrCreateDirectMessageChannel DMチャンネル取得する
func (repo *RepositoryImpl) GetDirectMessageChannel(user1, user2 uuid.UUID) (*model.Channel, error) {
	var channel model.Channel

	// ユーザーが存在するかどうかの判定はusers_private_channelsテーブルに外部キー制約が貼ってあるのでそれで対応する

	// チャンネル存在確認
	if user1 == user2 {
		// 自分宛DM
		err := repo.db.
			Where("parent_id = ? AND id IN ?", dmChannelRootUUID, repo.db.
				Table("users_private_channels").
				Select("channel_id").
				Group("channel_id").
				Having("COUNT(*) = 1 AND GROUP_CONCAT(user_id) = ?", user1).
				SubQuery()).
			Take(&channel).
			Error
		if err != nil {
			if !gorm.IsRecordNotFoundError(err) {
				return nil, err
			}
		} else {
			return &channel, nil
		}
	} else {
		// 他人宛DM
		err := repo.db.
			Where("parent_id = ? AND id IN ?", dmChannelRootUUID, repo.db.
				Raw("SELECT u.channel_id FROM users_private_channels AS u INNER JOIN (SELECT channel_id FROM users_private_channels GROUP BY channel_id HAVING COUNT(*) = 2) AS ex ON ex.channel_id = u.channel_id AND u.user_id IN (?, ?) GROUP BY channel_id HAVING COUNT(*) = 2", user1, user2).
				SubQuery()).
			Take(&channel).
			Error
		if err != nil {
			if !gorm.IsRecordNotFoundError(err) {
				return nil, err
			}
		} else {
			return &channel, nil
		}
	}

	// 存在しなかったので作成
	channel = model.Channel{
		ID:        uuid.NewV4(),
		Name:      "dm_" + utils.RandAlphabetAndNumberString(17),
		ParentID:  dmChannelRootUUID,
		IsPublic:  false,
		IsVisible: true,
		IsForced:  false,
	}

	err := repo.transact(func(tx *gorm.DB) error {
		if err := tx.Create(&channel).Error; err != nil {
			return err
		}

		// メンバーに追加
		if err := tx.Create(&model.UsersPrivateChannel{UserID: user1, ChannelID: channel.ID}).Error; err != nil {
			return err
		}
		if user1 != user2 {
			if err := tx.Create(&model.UsersPrivateChannel{UserID: user2, ChannelID: channel.ID}).Error; err != nil {
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

// GetAllChannels 全てのチャンネルを取得する
func (repo *RepositoryImpl) GetAllChannels() (channels []*model.Channel, err error) {
	channels = make([]*model.Channel, 0)
	err = repo.db.Find(&channels).Error
	return
}

// IsChannelPresent チャンネル名が同階層に既に存在するか
func (repo *RepositoryImpl) IsChannelPresent(name string, parent uuid.UUID) (bool, error) {
	c := 0
	err := repo.db.
		Model(&model.Channel{}).
		Where("parent_id = ? AND name = ?", parent, name).
		Limit(1).
		Count(&c).
		Error
	if err != nil {
		return false, err
	}

	return c > 0, nil
}

// IsChannelAccessibleToUser 指定したチャンネルが指定したユーザーからアクセス可能かどうか
func (repo *RepositoryImpl) IsChannelAccessibleToUser(userID, channelID uuid.UUID) (bool, error) {
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
	if err != nil {
		return false, err
	}
	return c > 0, nil
}

// GetParentChannel 親のチャンネルを取得する
func (repo *RepositoryImpl) GetParentChannel(channelID uuid.UUID) (*model.Channel, error) {
	if channelID == uuid.Nil {
		return nil, repository.ErrNotFound
	}

	var p []uuid.UUID
	err := repo.db.
		Model(&model.Channel{}).
		Where(&model.Channel{ID: channelID}).
		Pluck("parent_id", &p).
		Error
	if err != nil {
		return nil, err
	}
	if len(p) == 0 {
		return nil, repository.ErrNotFound
	} else if p[0] == uuid.Nil {
		return nil, nil
	}

	ch := &model.Channel{}
	err = repo.db.
		Where(&model.Channel{ID: p[0]}).
		Take(ch).
		Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return ch, nil
}

// GetChildrenChannelIDs 子チャンネルのIDを取得する
func (repo *RepositoryImpl) GetChildrenChannelIDs(channelID uuid.UUID) (children []uuid.UUID, err error) {
	children = make([]uuid.UUID, 0)
	if channelID == uuid.Nil {
		return children, nil
	}
	err = repo.db.
		Model(&model.Channel{}).
		Where(&model.Channel{ParentID: channelID}).
		Pluck("id", &children).Error
	return children, err
}

// GetDescendantChannelIDs 子孫チャンネルのIDを取得する
func (repo *RepositoryImpl) GetDescendantChannelIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	var descendants []uuid.UUID
	children, err := repo.GetChildrenChannelIDs(channelID)
	if err != nil {
		return nil, err
	}
	descendants = append(descendants, children...)
	for _, v := range children {
		sub, err := repo.GetDescendantChannelIDs(v)
		if err != nil {
			return nil, err
		}
		descendants = append(descendants, sub...)
	}
	return descendants, nil
}

// GetAscendantChannelIDs 祖先チャンネルのIDを取得する
func (repo *RepositoryImpl) GetAscendantChannelIDs(channelID uuid.UUID) ([]uuid.UUID, error) {
	var ascendants []uuid.UUID
	parent, err := repo.GetParentChannel(channelID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, nil
		}
		return nil, err
	} else if parent == nil {
		return []uuid.UUID{}, nil
	}
	ascendants = append(ascendants, parent.ID)
	sub, err := repo.GetAscendantChannelIDs(parent.ID)
	if err != nil {
		return nil, err
	}
	ascendants = append(ascendants, sub...)
	return ascendants, nil
}

// GetChannelPath チャンネルのパス文字列を取得する
func (repo *RepositoryImpl) GetChannelPath(id uuid.UUID) (string, error) {
	if id == uuid.Nil {
		return "", repository.ErrNotFound
	}
	v, ok := repo.ChannelPathMap.Load(id)
	if ok {
		return v.(string), nil
	}

	ch := &model.Channel{}
	if err := repo.db.Where(&model.Channel{ID: id}).Take(ch).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return "", repository.ErrNotFound
		}
		return "", err
	}

	var path string
	if pid := ch.ParentID; pid != uuid.Nil {
		parentPath, err := repo.GetChannelPath(pid)
		if err != nil && err != repository.ErrNotFound {
			return "", err
		}
		path = parentPath + "/" + ch.Name
	} else {
		path = ch.Name
	}

	repo.ChannelPathMap.Store(id, path)
	return path, nil
}

// GetChannelDepth 指定したチャンネル木の深さを取得する
func (repo *RepositoryImpl) GetChannelDepth(id uuid.UUID) (int, error) {
	children, err := repo.GetChildrenChannelIDs(id)
	if err != nil {
		return 0, err
	}
	max := 0
	for _, v := range children {
		d, err := repo.GetChannelDepth(v)
		if err != nil {
			return 0, err
		}
		if max < d {
			max = d
		}
	}
	return max + 1, nil
}

// AddPrivateChannelMember プライベートチャンネルにメンバーを追加します
func (repo *RepositoryImpl) AddPrivateChannelMember(channelID, userID uuid.UUID) error {
	if channelID == uuid.Nil || userID == uuid.Nil {
		return repository.ErrNilID
	}
	var s model.UsersPrivateChannel
	return repo.db.FirstOrCreate(&s, &model.UsersPrivateChannel{UserID: userID, ChannelID: channelID}).Error
}

// GetPrivateChannelMemberIDs プライベートチャンネルのメンバーの配列を取得する
func (repo *RepositoryImpl) GetPrivateChannelMemberIDs(channelID uuid.UUID) (users []uuid.UUID, err error) {
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

// IsUserPrivateChannelMember ユーザーがプライベートチャンネルのメンバーかどうかを確認します
func (repo *RepositoryImpl) IsUserPrivateChannelMember(channelID, userID uuid.UUID) (bool, error) {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return false, nil
	}

	c := 0
	err := repo.db.
		Model(&model.UsersPrivateChannel{}).
		Where(&model.UsersPrivateChannel{ChannelID: channelID, UserID: userID}).
		Limit(1).
		Count(&c).
		Error
	if err != nil {
		return false, err
	}
	return c > 0, nil
}

// SubscribeChannel 指定したチャンネルを購読します
func (repo *RepositoryImpl) SubscribeChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	var s model.UserSubscribeChannel
	return repo.db.FirstOrCreate(&s, &model.UserSubscribeChannel{UserID: userID, ChannelID: channelID}).Error
}

// UnsubscribeChannel 指定したチャンネルの購読を解除します
func (repo *RepositoryImpl) UnsubscribeChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return nil
	}
	return repo.db.
		Where(&model.UserSubscribeChannel{UserID: userID, ChannelID: channelID}).
		Delete(&model.UserSubscribeChannel{}).
		Error
}

// GetSubscribingUserIDs 指定したチャンネルを購読しているユーザーを取得
func (repo *RepositoryImpl) GetSubscribingUserIDs(channelID uuid.UUID) (users []uuid.UUID, err error) {
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

// GetSubscribedChannelIDs ユーザーが購読しているチャンネルを取得する
func (repo *RepositoryImpl) GetSubscribedChannelIDs(userID uuid.UUID) (channels []uuid.UUID, err error) {
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
