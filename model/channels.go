package model

import (
	"errors"
	"github.com/jinzhu/gorm"
	"sync"
	"time"

	"github.com/traPtitech/traQ/utils/validator"

	"github.com/satori/go.uuid"
)

var (
	channelPathMap = sync.Map{}
	// ErrChannelPathDepth 作成されるチャンネルの深さが5より大きいときに返すエラー
	ErrChannelPathDepth = errors.New("channel depth is no more than 5")
	// ErrDuplicateName 作成されるチャンネルと同名のチャンネルが既に同階層に存在する場合に返すエラー
	ErrDuplicateName = errors.New("this name channel already exists")
)

// Channel :チャンネルの構造体
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
	ch.ID = CreateUUID()
	ch.IsVisible = true
	ch.UpdaterID = ch.CreatorID
	if err := ch.Validate(); err != nil {
		return err
	}

	// 階層チェック
	// FIXME 親チャンネルの存在を確認する
	// FIXME プライベートチャンネルの事を考える
	// 五階層までは許すけどそれ以上はダメ
	if len(ch.ParentID) == 36 {
		//ルートチャンネルではない
		ch, err := GetParentChannel(uuid.FromStringOrNil(ch.ParentID))
		if err != nil && err != ErrNotFound {
			return err
		}

		for i := 0; ; i++ {
			if ch == nil {
				if i >= 4 {
					return ErrChannelPathDepth
				}
				break
			}
			ch, err = GetParentChannel(ch.GetCID())
			if err != nil && err != ErrNotFound {
				return err
			}
		}
	}

	// チャンネル名重複を確認
	has, err := IsChannelNamePresent(ch.Name, ch.ParentID)
	if err != nil {
		return err
	}
	if has {
		return ErrDuplicateName
	}

	return nil
}

// Validate 構造体を検証します
func (ch *Channel) Validate() error {
	return validator.ValidateStruct(ch)
}

// GetCID チャンネルのUUIDを返します
func (ch *Channel) GetCID() uuid.UUID {
	return uuid.Must(uuid.FromString(ch.ID))
}

// GetCreatorID チャンネル作成者のUUIDを返します
func (ch *Channel) GetCreatorID() uuid.UUID {
	return uuid.Must(uuid.FromString(ch.CreatorID))
}

// CreateChannel チャンネルを作成します
func CreateChannel(parent, name string, creatorID uuid.UUID, isPublic bool) (*Channel, error) {
	ch := &Channel{
		Name:      name,
		ParentID:  parent,
		CreatorID: creatorID.String(),
		IsPublic:  isPublic,
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

// UpdateChannelTopic チャンネルトピックを更新します
func UpdateChannelTopic(channelID uuid.UUID, topic string, updaterID uuid.UUID) error {
	return db.Model(Channel{ID: channelID.String()}).Updates(map[string]interface{}{
		"topic":      topic,
		"updater_id": updaterID.String(),
	}).Error
}

// ChangeChannelName チャンネル名を変更します
func ChangeChannelName(channelID uuid.UUID, name string, updaterID uuid.UUID) error {
	if err := validator.ValidateVar(name, "channel,required"); err != nil {
		return err
	}

	ch, err := GetChannel(channelID)
	if err != nil {
		return err
	}

	// チャンネル名重複を確認
	has, err := IsChannelNamePresent(name, ch.ParentID)
	if err != nil {
		return err
	}
	if has {
		return ErrDuplicateName
	}

	err = db.Model(Channel{ID: channelID.String()}).Updates(map[string]interface{}{
		"name":       name,
		"updater_id": updaterID.String(),
	}).Error
	if err != nil {
		return err
	}

	//チャンネルパスキャッシュの更新
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

	// 階層チェック
	// FIXME 変更するチャンネルの子の事を考えてない
	// FIXME 循環参照を考えてない
	// FIXME プライベートチャンネルの事を考えてない
	// FIXME 変更先の存在を確認してない
	// 五階層までは許すけどそれ以上はダメ
	if len(parent) == 36 {
		//ルートチャンネルではない
		ch, err := GetParentChannel(uuid.FromStringOrNil(parent))
		if err != nil && err != ErrNotFound {
			return err
		}

		for i := 0; ; i++ {
			if ch == nil {
				if i >= 4 {
					return ErrChannelPathDepth
				}
				break
			}
			ch, err = GetParentChannel(ch.GetCID())
			if err != nil && err != ErrNotFound {
				return err
			}
		}
	}

	// チャンネル名重複を確認
	has, err := IsChannelNamePresent(ch.Name, parent)
	if err != nil {
		return err
	}
	if has {
		return ErrDuplicateName
	}

	err = db.Model(Channel{ID: channelID.String()}).Updates(map[string]interface{}{
		"parent_id":  parent,
		"updater_id": updaterID.String(),
	}).Error
	if err != nil {
		return err
	}

	//チャンネルパスキャッシュの更新
	ch.ParentID = parent
	updateChannelPathWithDescendants(ch)

	return nil
}

// UpdateChannelFlag チャンネルの各種フラグを更新します
func UpdateChannelFlag(channelID uuid.UUID, visibility, forced *bool, updaterID uuid.UUID) error {
	data := map[string]interface{}{
		"updater_id": updaterID.String(),
	}
	if visibility != nil {
		data["is_visible"] = *visibility
	}
	if forced != nil {
		data["is_forced"] = *forced
	}

	return db.Model(Channel{ID: channelID.String()}).Updates(data).Error
}

// DeleteChannel チャンネルを削除します
func DeleteChannel(channelID uuid.UUID) error {
	return db.Delete(Channel{ID: channelID.String()}).Error
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
	p := &Channel{}
	err := db.
		Model(Channel{}).
		Select("parent_id").
		Where("id = ?", channelID.String()).
		Take(p).
		Error
	if err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if len(p.ParentID) == 0 {
		return nil, nil
	}

	ch := &Channel{}
	err = db.
		Where("id = ?", p.ParentID).
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

// GetChildrenChannelIDsWithUserID userIDのユーザーから見えるchannelIDの子チャンネルのIDを取得する
func GetChildrenChannelIDsWithUserID(userID uuid.UUID, channelID string) (children []string, err error) {
	err = db.
		Model(Channel{}).
		Joins("LEFT JOIN users_private_channels ON users_private_channels.channel_id = channels.id").
		Where("(channels.is_public = true OR users_private_channels.user_id = ?) AND channels.parent_id = ?", userID, channelID).
		Pluck("channels.id", &children).
		Error
	return
}

// GetChildrenChannelIDs 子チャンネルのIDを取得する
func GetChildrenChannelIDs(channelID uuid.UUID) (children []string, err error) {
	err = db.Model(Channel{}).Where(&Channel{ParentID: channelID.String()}).Pluck("id", &children).Error
	return
}

// GetChannel チャンネルを取得する
func GetChannel(channelID uuid.UUID) (*Channel, error) {
	ch := &Channel{}
	err := db.Where("id = ?", channelID.String()).Take(ch).Error
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
		Where("id IN (?)", db.Model(Message{}).Select("messages.channel_id").Where(Message{ID: messageID.String()}).QueryExpr()).
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
		Where("channels.is_public = true OR users_private_channels.user_id = ?", userID.String()).
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

func updateChannelPathWithDescendants(channel *Channel) error {
	path, err := channel.Path()
	if err != nil {
		return err
	}

	channelPathMap.Store(channel.GetCID(), path)

	//子チャンネルも
	var children []*Channel
	if err = db.Find(&children, Channel{ParentID: channel.ID}).Error; err != nil {
		return err
	}

	for _, v := range children {
		if err := updateChannelPathWithDescendants(v); err != nil {
			return err
		}
	}

	return nil
}
