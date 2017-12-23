package model

import (
	"fmt"
	"regexp"
)

// Channel :チャンネルの構造体
type Channel struct {
	ID        string `xorm:"char(36) pk"`
	Name      string `xorm:"varchar(20) not null"`
	ParentID  string `xorm:"parent_id char(36) not null"`
	Topic     string `xorm:"text"`
	IsForced  bool   `xorm:"bool not null"`
	IsDeleted bool   `xorm:"bool not null"`
	IsPublic  bool   `xorm:"bool not null"`
	IsVisible bool   `xorm:"bool not null"`
	CreatorID string `xorm:"char(36) not null"`
	CreatedAt int    `xorm:"created not null"`
	UpdaterID string `xorm:"char(36) not null"`
	UpdatedAt int    `xorm:"updated not null"`
}

// TableName : テーブル名を指定するメソッド
func (channel *Channel) TableName() string {
	return "channels"
}

// Create : チャンネル作成を行うメソッド
func (channel *Channel) Create() error {
	if channel.ID != "" {
		return fmt.Errorf("ID is not empty! You can use Update()")
	}

	if channel.Name == "" {
		return fmt.Errorf("Name is empty")
	}

	if channel.CreatorID == "" {
		return fmt.Errorf("CreatorID is empty")
	}

	if err := validateChannelName(channel.Name); err != nil {
		return err
	}

	channel.ID = CreateUUID()
	channel.IsVisible = true

	channel.UpdaterID = channel.CreatorID

	if _, err := db.Insert(channel); err != nil {
		return fmt.Errorf("Failed to create channel: %v", err)
	}
	return nil
}

// GetChannelByID : チャンネルIDによってチャンネルを取得
func GetChannelByID(userID, channelID string) (*Channel, error) {
	channel := &Channel{}
	channel.ID = channelID
	has, err := db.Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("(is_public = true OR user_id = ?) AND is_deleted = false", userID).Get(channel)

	if err != nil {
		return nil, fmt.Errorf("Failed to get channel: %v", err)
	}

	if !has {
		return nil, fmt.Errorf("指定されたチャンネルは存在しないかユーザーからは見ることができません")
	}

	return channel, nil
}

// Exists : 指定したチャンネルがuserIDのユーザーから見えるチャンネルかどうかを確認する
func (channel *Channel) Exists(userID string) (bool, error) {
	if userID != "" {
		has, err := db.Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("(is_public = true OR user_id = ?) AND is_deleted = false", userID).Get(channel)
		return has, err
	}
	return db.Get(channel)
}

// GetChannels : userIDのユーザーから見えるチャンネルの一覧を取得する
func GetChannels(userID string) ([]*Channel, error) {
	// TODO: 隠しチャンネルを表示するかどうかをクライアントと決める
	var channelList []*Channel
	err := db.Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("(is_public = true OR user_id = ?) AND is_deleted = false", userID).Find(&channelList)

	if err != nil {
		return nil, fmt.Errorf("Failed to find channels: %v", err)
	}
	return channelList, nil
}

// Children :  userIDのユーザーから見えるchannelIDの子チャンネル
func (channel *Channel) Children(userID string) ([]string, error) {
	var channelIDList []string
	if channel.ID == "" {
		return nil, fmt.Errorf("Channel ID is empty")
	}
	err := db.Table("channels").Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("(is_public = true OR user_id = ?) AND parent_id = ? AND is_deleted = false", userID, channel.ID).Cols("id").Find(&channelIDList)

	if err != nil {
		return nil, fmt.Errorf("Failed to find channels: %v", err)
	}
	return channelIDList, nil
}

// Update : チャンネルの情報の更新を行う
func (channel *Channel) Update() error {
	_, err := db.ID(channel.ID).UseBool().Update(channel)
	if err != nil {
		return fmt.Errorf("Failed to update channel: %v", err)
	}
	return nil
}

func validateChannelName(name string) error {
	if !regexp.MustCompile(`^[a-zA-Z0-9-_]*$`).Match([]byte(name)) {
		return fmt.Errorf("使用できる文字は半角英数字と-_のみです")
	}

	if len(name) > 20 {
		return fmt.Errorf("チャンネル名は最大20文字です")
	}
	return nil
}
