package model

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/utils"
	"sync"
	"time"

	"github.com/traPtitech/traQ/utils/validator"

	"github.com/satori/go.uuid"
)

const (
	directMessageChannelRootID = "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"
	maxChannelDepth            = 5
)

var (
	channelPathMap = sync.Map{}
	// ErrChannelDepthLimitation チャンネルの深さが5より大きくなる
	ErrChannelDepthLimitation = fmt.Errorf("channel depth　must be <= %d", maxChannelDepth)
	// ErrDuplicateName 作成されるチャンネルと同名のチャンネルが既に同階層に存在する
	ErrDuplicateName = errors.New("this name's channel already exists")
	// ErrParentChannelDifferentOpenStatus 作成されるチャンネルが親チャンネルの公開状況と異なる
	ErrParentChannelDifferentOpenStatus = errors.New("the private channel's parent must not be public and vice versa")
	// ErrDirectMessageChannelIsOpen ダイレクトメッセージチャンネルはpublicに出来ない
	ErrDirectMessageChannelIsOpen = errors.New("direct message channel must be private")
	// ErrDirectMessageChannelCannotHaveChildren ダイレクトメッセージチャンネルは子を持てない
	ErrDirectMessageChannelCannotHaveChildren = errors.New("direct message channel cannot have children")
)

// Channel チャンネルの構造体
type Channel struct {
	ID        string `gorm:"type:char(36);primary_key"                 validate:"uuid,required"`
	Name      string `gorm:"type:varchar(20);unique_index:name_parent" validate:"channel,required"`
	ParentID  string `gorm:"type:char(36);unique_index:name_parent"`
	Topic     string `gorm:"type:text"`
	IsForced  bool
	IsPublic  bool
	IsVisible bool
	CreatorID string     `gorm:"type:char(36)"                             validate:"uuid,required"`
	UpdaterID string     `gorm:"type:char(36)"                             validate:"uuid,required"`
	CreatedAt time.Time  `gorm:"precision:6"`
	UpdatedAt time.Time  `gorm:"precision:6"`
	DeletedAt *time.Time `gorm:"precision:6"`
}

// TableName テーブル名を指定するメソッド
func (ch *Channel) TableName() string {
	return "channels"
}

// BeforeCreate db.Create前に呼び出されます
func (ch *Channel) BeforeCreate(tx *gorm.DB) error {
	if len(ch.ID) != 36 {
		ch.ID = CreateUUID()
	}
	if len(ch.UpdaterID) != 36 {
		ch.UpdaterID = ch.CreatorID
	}
	ch.IsVisible = true
	return ch.CheckConsistency()
}

// CheckConsistency チャンネルの一貫性を検証します。チャンネル作成・移動前に呼び出し
func (ch *Channel) CheckConsistency() error {
	if err := validator.ValidateStruct(ch); err != nil {
		return err
	}

	switch ch.ParentID {
	case "": // ルート

	case directMessageChannelRootID: // DMルート
		if ch.IsPublic {
			return ErrDirectMessageChannelIsOpen
		}

	default: // ルート以外
		// 親の存在を確認
		parentCh, err := GetChannel(uuid.FromStringOrNil(ch.ParentID))
		if err != nil {
			return err
		}

		// DMチャンネルの子チャンネルには出来ない
		if parentCh.IsDMChannel() {
			return ErrDirectMessageChannelCannotHaveChildren
		}

		// 親と公開状況が一致しているか
		if ch.IsPublic != parentCh.IsPublic {
			return ErrParentChannelDifferentOpenStatus
		}

		// 深さを検証
		depth := 1
		for { // 祖先
			if len(parentCh.ParentID) == 0 {
				break
			}
			if ch.GetCID() == parentCh.GetCID() {
				// ループ検出
				return ErrChannelDepthLimitation
			}

			parentCh, err = GetChannel(uuid.FromStringOrNil(parentCh.ParentID))
			if err != nil {
				if err == ErrNotFound {
					break
				}
				return err
			}
			depth++
			if depth > maxChannelDepth {
				break
			}
		}
		bottom, err := GetChannelDepth(ch.GetCID()) // 子孫
		if err != nil {
			return err
		}
		depth += bottom
		if depth > maxChannelDepth {
			return ErrChannelDepthLimitation
		}
	}

	// チャンネル名検証
	has, err := IsChannelNamePresent(ch.Name, ch.ParentID)
	if err != nil {
		return err
	}
	if has {
		return ErrDuplicateName
	}

	return nil
}

// GetCID チャンネルのUUIDを返します
func (ch *Channel) GetCID() uuid.UUID {
	return uuid.Must(uuid.FromString(ch.ID))
}

// GetCreatorID チャンネル作成者のUUIDを返します
func (ch *Channel) GetCreatorID() uuid.UUID {
	return uuid.Must(uuid.FromString(ch.CreatorID))
}

// IsDMChannel ダイレクトメッセージ用チャンネルかどうかを返します
func (ch *Channel) IsDMChannel() bool {
	return ch.ParentID == directMessageChannelRootID
}

// Path チャンネルのパス文字列を取得する
func (ch *Channel) Path() (string, error) {
	path := ch.Name
	current := ch

	for {
		parent, err := GetParentChannel(current.GetCID())
		if err != nil {
			return "#" + path, nil
		}
		if parent == nil {
			break
		}

		if parentPath, ok := GetChannelPath(uuid.FromStringOrNil(parent.ID)); ok {
			return parentPath + "/" + path, nil
		}

		path = parent.Name + "/" + path
		current = parent
	}

	return "#" + path, nil
}

// CreatePublicChannel パブリックチャンネルを作成します
func CreatePublicChannel(parent, name string, creatorID uuid.UUID) (*Channel, error) {
	ch := &Channel{
		Name:      name,
		ParentID:  parent,
		CreatorID: creatorID.String(),
		IsPublic:  true,
	}

	if err := db.Create(ch).Error; err != nil {
		return nil, err
	}

	// チャンネルパスをキャッシュ
	if path, err := ch.Path(); err == nil {
		channelPathMap.Store(ch.GetCID(), path)
	}

	return ch, nil
}

// CreatePrivateChannel プライベートチャンネルを作成します
func CreatePrivateChannel(parent, name string, creatorID uuid.UUID, members []uuid.UUID) (*Channel, error) {
	if parent == directMessageChannelRootID {
		return nil, ErrForbidden // GetOrCreateDirectMessageChannelを使え
	}

	validMember := make([]uuid.UUID, 0, len(members))
	for _, v := range members {
		ok, err := UserExists(v)
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

	ch := &Channel{
		Name:      name,
		ParentID:  parent,
		CreatorID: creatorID.String(),
		IsPublic:  false,
		IsForced:  false,
		IsVisible: true,
	}

	// TODO 親チャンネルのメンバーと比較検証

	err := transact(func(tx *gorm.DB) error {
		if err := db.Create(ch).Error; err != nil {
			return err
		}

		// メンバーに追加
		for _, v := range validMember {
			if err := tx.Create(&UsersPrivateChannel{UserID: v, ChannelID: ch.GetCID()}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// チャンネルパスをキャッシュ
	if path, err := ch.Path(); err == nil {
		channelPathMap.Store(ch.GetCID(), path)
	}

	return ch, nil
}

// GetOrCreateDirectMessageChannel DMチャンネルが存在すればそれを返し、存在しなければ作成します
func GetOrCreateDirectMessageChannel(user1, user2 uuid.UUID) (*Channel, error) {
	var channel Channel

	// チャンネル存在確認
	if user1 == user2 {
		// 自分宛DM
		err := db.
			Where("parent_id = ? AND id IN ?", directMessageChannelRootID, db.
				Select("channel_id").
				Group("channel_id").
				Having("COUNT(*) = 1 AND GROUP_CONCAT(user_id) = ?", user1).
				Limit(1).SubQuery()).
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
		err := db.
			Where("parent_id = ? AND id IN ?", directMessageChannelRootID, db.
				Raw("SELECT u.channel_id FROM users_private_channels AS u INNER JOIN (SELECT channel_id FROM users_private_channels GROUP BY channel_id HAVING COUNT(*) = 2) AS ex ON ex.channel_id = u.channel_id AND u.user_id IN (?, ?) GROUP BY channel_id HAVING COUNT(*) = 2 LIMIT 1", user1, user2).
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
	channel = Channel{
		Name:      "dm_" + utils.RandAlphabetAndNumberString(17),
		ParentID:  directMessageChannelRootID,
		CreatorID: serverUser.ID,
		IsPublic:  false,
		IsVisible: true,
		IsForced:  false,
	}

	err := transact(func(tx *gorm.DB) error {
		if err := tx.Create(&channel).Error; err != nil {
			return err
		}

		// メンバーに追加
		if err := tx.Create(&UsersPrivateChannel{UserID: user1, ChannelID: channel.GetCID()}).Error; err != nil {
			return err
		}
		if user1 != user2 {
			if err := tx.Create(&UsersPrivateChannel{UserID: user2, ChannelID: channel.GetCID()}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// TODO イベント発行
	return &channel, nil
}

// UpdateChannelTopic チャンネルトピックを更新します
func UpdateChannelTopic(channelID uuid.UUID, topic string, updaterID uuid.UUID) error {
	return db.Model(&Channel{ID: channelID.String()}).Updates(map[string]interface{}{
		"topic":      topic,
		"updater_id": updaterID,
	}).Error
}

// ChangeChannelName チャンネル名を変更します
func ChangeChannelName(channelID uuid.UUID, name string, updaterID uuid.UUID) error {
	// チャンネル名検証
	if err := validator.ValidateVar(name, "channel,required"); err != nil {
		return err
	}

	// チャンネル取得
	ch, err := GetChannel(channelID)
	if err != nil {
		return err
	}

	// ダイレクトメッセージチャンネルの名前は変更不可能
	if ch.IsDMChannel() {
		return ErrForbidden
	}

	// チャンネル名重複を確認
	has, err := IsChannelNamePresent(name, ch.ParentID)
	if err != nil {
		return err
	}
	if has {
		return ErrDuplicateName
	}

	// 更新
	err = db.Model(&Channel{ID: channelID.String()}).Updates(map[string]interface{}{
		"name":       name,
		"updater_id": updaterID,
	}).Error
	if err != nil {
		return err
	}

	// チャンネルパスキャッシュの更新
	ch.Name = name
	updateChannelPathWithDescendants(ch)

	return nil
}

// ChangeChannelParent チャンネルの親を変更します
func ChangeChannelParent(channelID uuid.UUID, parent string, updaterID uuid.UUID) error {
	ch, err := GetChannel(channelID)
	if err != nil {
		return err
	}

	// ダイレクトメッセージチャンネルの親は変更不可能
	if ch.IsDMChannel() {
		return ErrForbidden
	}

	ch.ParentID = parent
	if err := ch.CheckConsistency(); err != nil {
		return err
	}

	err = db.Model(&Channel{ID: channelID.String()}).Updates(map[string]interface{}{
		"parent_id":  parent,
		"updater_id": updaterID,
	}).Error
	if err != nil {
		return err
	}

	//チャンネルパスキャッシュの更新
	updateChannelPathWithDescendants(ch)

	return nil
}

// UpdateChannelFlag チャンネルの各種フラグを更新します
func UpdateChannelFlag(channelID uuid.UUID, visibility, forced *bool, updaterID uuid.UUID) error {
	data := map[string]interface{}{
		"updater_id": updaterID,
	}
	if visibility != nil {
		data["is_visible"] = *visibility
	}
	if forced != nil {
		data["is_forced"] = *forced
	}

	return db.Model(&Channel{ID: channelID.String()}).Updates(data).Error
}

// DeleteChannel 子孫チャンネルを含めてチャンネルを削除します
func DeleteChannel(channelID uuid.UUID) error {
	desc, err := GetDescendantChannelIDs(channelID)
	if err != nil {
		return err
	}
	desc = append(desc, channelID)

	err = transact(func(tx *gorm.DB) error {
		for _, v := range desc {
			if err := tx.Delete(&Channel{ID: v.String()}).Error; err != nil {
				return err
			}
		}
		return nil
	})

	// TODO イベント発行
	return err
}

// IsChannelNamePresent チャンネル名が同階層に既に存在するか
func IsChannelNamePresent(name, parent string) (bool, error) {
	c := 0
	err := db.
		Model(Channel{}).
		Where("parent_id = ? AND name = ?", parent, name).
		Limit(1).
		Count(&c).
		Error
	if err != nil {
		return false, err
	}

	return c > 0, nil
}

// GetParentChannel 親のチャンネルを取得する
func GetParentChannel(channelID uuid.UUID) (*Channel, error) {
	p := ""
	err := db.
		Model(Channel{}).
		Where("id = ?", channelID).
		Pluck("parent_id", &p).
		Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if len(p) == 0 {
		return nil, nil
	}

	ch := &Channel{}
	err = db.
		Where("id = ?", p).
		Take(ch).
		Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return ch, nil
}

// GetChildrenChannelIDs 子チャンネルのIDを取得する
func GetChildrenChannelIDs(channelID uuid.UUID) (children []uuid.UUID, err error) {
	err = db.Model(Channel{}).Where(&Channel{ParentID: channelID.String()}).Pluck("id", &children).Error
	return
}

// GetDescendantChannelIDs 子孫チャンネルのIDを取得する
func GetDescendantChannelIDs(channelID uuid.UUID) (descendants []uuid.UUID, err error) {
	children, err := GetChildrenChannelIDs(channelID)
	if err != nil {
		return nil, err
	}
	for _, v := range children {
		sub, err := GetDescendantChannelIDs(v)
		if err != nil {
			return nil, err
		}
		descendants = append(descendants, sub...)
	}
	return descendants, nil
}

// GetAscendantChannelIDs 祖先チャンネルのIDを取得する
func GetAscendantChannelIDs(channelID uuid.UUID) (ascendants []uuid.UUID, err error) {
	parent, err := GetParentChannel(channelID)
	if err != nil {
		if err == ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	ascendants = append(ascendants, parent.GetCID())
	sub, err := GetAscendantChannelIDs(parent.GetCID())
	if err != nil {
		return nil, err
	}
	ascendants = append(ascendants, sub...)
	return ascendants, nil
}

// GetChannel チャンネルを取得する
func GetChannel(channelID uuid.UUID) (*Channel, error) {
	ch := &Channel{}
	err := db.Where("id = ?", channelID).Take(ch).Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return ch, nil
}

// GetChannelWithUserID 指定したチャンネルが指定したユーザーがアクセス可能な場合チャンネルを取得
func GetChannelWithUserID(userID, channelID uuid.UUID) (*Channel, error) {
	channel := &Channel{}

	err := db.
		Joins("LEFT JOIN users_private_channels ON users_private_channels.channel_id = channels.id").
		Where("(channels.is_public = true OR users_private_channels.user_id = ?) AND channels.id = ?", userID, channelID).
		Take(channel).
		Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFoundOrForbidden
		}
		return nil, err
	}

	return channel, nil
}

// IsChannelAccessibleToUser 指定したチャンネルが指定したユーザーからアクセス可能かどうか
func IsChannelAccessibleToUser(userID, channelID uuid.UUID) (bool, error) {
	c := 0
	err := db.
		Model(Channel{}).
		Joins("LEFT JOIN users_private_channels ON users_private_channels.channel_id = channels.id").
		Where("(channels.is_public = true OR users_private_channels.user_id = ?) AND channels.id = ?", userID, channelID).
		Count(&c).
		Error
	if err != nil {
		return false, err
	}

	return c > 0, nil
}

// GetChannelByMessageID メッセージIDによってチャンネルを取得
func GetChannelByMessageID(messageID uuid.UUID) (*Channel, error) {
	channel := &Channel{}

	err := db.
		Where("id IN (?)", db.Model(Message{}).Select("messages.channel_id").Where(&Message{ID: messageID.String()}).QueryExpr()).
		Take(channel).
		Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return channel, nil
}

// GetChannelList userIDのユーザーから見えるチャンネルの一覧を取得する
func GetChannelList(userID uuid.UUID) (channels []*Channel, err error) {
	err = db.
		Joins("LEFT JOIN users_private_channels ON users_private_channels.channel_id = channels.id").
		Where("channels.is_public = true OR users_private_channels.user_id = ?", userID).
		Find(&channels).
		Error
	return
}

// GetAllChannels 全てのチャンネルを取得する
func GetAllChannels() (channels []*Channel, err error) {
	err = db.Find(&channels).Error
	return
}

// GetChannelPath 指定したIDのチャンネルのパス文字列を取得する
func GetChannelPath(id uuid.UUID) (string, bool) {
	v, ok := channelPathMap.Load(id)
	if !ok {
		return "", false
	}

	return v.(string), true
}

// GetChannelDepth 指定したチャンネル木の深さを取得する
func GetChannelDepth(id uuid.UUID) (int, error) {
	children, err := GetChildrenChannelIDs(id)
	if err != nil {
		return 0, err
	}
	max := -1
	for _, v := range children {
		d, err := GetChannelDepth(v)
		if err != nil {
			return 0, err
		}
		if max < d {
			max = d
		}
	}
	return max + 1, nil
}

func updateChannelPathWithDescendants(channel *Channel) error {
	path, err := channel.Path()
	if err != nil {
		return err
	}

	channelPathMap.Store(channel.GetCID(), path)

	//子チャンネルも
	var children []*Channel
	if err = db.Find(&children, &Channel{ParentID: channel.ID}).Error; err != nil {
		return err
	}

	for _, v := range children {
		if err := updateChannelPathWithDescendants(v); err != nil {
			return err
		}
	}

	return nil
}
