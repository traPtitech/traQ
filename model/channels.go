package model

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/traPtitech/traQ/utils/validator"

	"github.com/satori/go.uuid"
)

var (
	channelPathMap = &sync.Map{}
	// ErrChannelPathDepth 作成されるチャンネルの深さが5より大きいときに返すエラー
	ErrChannelPathDepth = errors.New("Channel depth is no more than 5")
)

// Channel :チャンネルの構造体
type Channel struct {
	ID        string    `xorm:"char(36) pk"                                     validate:"uuid,required"`
	Name      string    `xorm:"varchar(20) not null unique(name_parent)"        validate:"channel,required"`
	ParentID  string    `xorm:"parent_id char(36) not null unique(name_parent)"`
	Topic     string    `xorm:"text"`
	IsForced  bool      `xorm:"bool not null"`
	IsDeleted bool      `xorm:"bool not null"`
	IsPublic  bool      `xorm:"bool not null"`
	IsVisible bool      `xorm:"bool not null"`
	CreatorID string    `xorm:"char(36) not null"                               validate:"uuid,required"`
	CreatedAt time.Time `xorm:"created not null"`
	UpdaterID string    `xorm:"char(36) not null"                               validate:"uuid,required"`
	UpdatedAt time.Time `xorm:"updated not null"`
}

// TableName テーブル名を指定するメソッド
func (channel *Channel) TableName() string {
	return "channels"
}

// Validate 構造体を検証します
func (channel *Channel) Validate() error {
	return validator.ValidateStruct(channel)
}

// Create チャンネル作成を行うメソッド
func (channel *Channel) Create() error {
	if channel.ID != "" {
		return fmt.Errorf("ID is not empty! You can use Update()")
	}

	channel.ID = CreateUUID()
	channel.IsVisible = true
	channel.UpdaterID = channel.CreatorID

	if err := channel.Validate(); err != nil {
		return err
	}

	// 階層チェック
	// 五階層までは許すけどそれ以上はダメ
	ch, err := channel.Parent()
	for i := 0; ; i++ {
		if ch == nil {
			if i >= 5 {
				return ErrChannelPathDepth
			}
			break
		}
		if err != nil {
			return err // NotFoundの場合はch == nil => true なのでここに到達しない
		}
		ch, err = ch.Parent()
	}

	// ここまでで入力されない要素は初期値(""や0)で格納される
	if _, err := db.Insert(channel); err != nil {
		return err
	}

	//チャンネルパスをキャッシュ
	if path, err := channel.Path(); err == nil {
		channelPathMap.Store(uuid.FromStringOrNil(channel.ID), path)
	}

	return nil
}

// Exists 指定したチャンネルがuserIDのユーザーから見えるチャンネルかどうかを確認する
func (channel *Channel) Exists(userID string) (bool, error) {
	if userID != "" {
		has, err := db.Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("(is_public = true OR user_id = ?) AND is_deleted = false", userID).Get(channel)
		return has, err
	}
	return db.Get(channel)
}

// Update チャンネルの情報の更新を行う
func (channel *Channel) Update() error {
	if err := channel.Validate(); err != nil {
		return err
	}

	_, err := db.ID(channel.ID).UseBool().MustCols().Update(channel)
	if err != nil {
		return err
	}

	//チャンネルパスキャッシュの更新
	updateChannelPathWithDescendants(channel)

	return nil
}

// Parent 親チャンネルを取得する
func (channel *Channel) Parent() (*Channel, error) {
	if len(channel.ParentID) == 0 {
		return nil, nil
	}

	parent := &Channel{}
	has, err := db.Where("id = ?", channel.ParentID).Get(parent)
	if !has {
		return nil, ErrNotFound
	}
	return parent, err
}

// Children userIDのユーザーから見えるchannelIDの子チャンネル
func (channel *Channel) Children(userID string) ([]string, error) {
	var channelIDList []string
	if channel.ID == "" {
		return nil, fmt.Errorf("channelID is empty")
	}
	err := db.Table("channels").Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("(is_public = true OR user_id = ?) AND parent_id = ? AND is_deleted = false", userID, channel.ID).Cols("id").Find(&channelIDList)
	if err != nil {
		return nil, err
	}
	return channelIDList, nil
}

// Path チャンネルのパス文字列を取得する
func (channel *Channel) Path() (string, error) {
	path := channel.Name
	current := channel

	for {
		parent, err := current.Parent()
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

// GetChannelByID チャンネルIDによってチャンネルを取得
func GetChannelByID(userID, channelID string) (*Channel, error) {
	channel := &Channel{}
	channel.ID = channelID

	has, err := db.Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("(is_public = true OR user_id = ?) AND is_deleted = false", userID).Get(channel)
	if err != nil {
		return nil, err
	}

	if !has {
		return nil, ErrNotFoundOrForbidden
	}

	return channel, nil
}

// GetChannelByMessageID メッセージIDによってチャンネルを取得
// チャンネルがis_deletedでも取得可能
func GetChannelByMessageID(messageID string) (*Channel, error) {
	channel := &Channel{}

	has, err := db.Join("INNER", "messages", "messages.channel_id = channels.id").Where("messages.id = ?", messageID).Get(channel)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrNotFound
	}

	return channel, nil
}

// GetChannelList userIDのユーザーから見えるチャンネルの一覧を取得する
func GetChannelList(userID string) ([]*Channel, error) {
	// TODO: 隠しチャンネルを表示するかどうかをクライアントと決める
	var channelList []*Channel
	err := db.Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("(is_public = true OR user_id = ?) AND is_deleted = false", userID).Find(&channelList)
	if err != nil {
		return nil, err
	}
	return channelList, nil
}

// GetAllChannels 全てのチャンネルを取得する
func GetAllChannels() (channels []*Channel, err error) {
	err = db.Find(&channels)
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

	channelPathMap.Store(uuid.FromStringOrNil(channel.ID), path)

	//子チャンネルも
	var children []*Channel
	if err = db.Where("parent_id = ?", channel.ID).Find(&children); err != nil {
		return err
	}

	for _, v := range children {
		if err := updateChannelPathWithDescendants(v); err != nil {
			return err
		}
	}

	return nil
}
