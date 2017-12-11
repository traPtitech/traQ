package model

import "fmt"

// Channel :チャンネルの構造体
type Channel struct {
	ID        string `xorm:"id char(36) pk"`
	Name      string `xorm:"varchar(20) not null"`
	ParentID  string `xorm:"parent_id char(36) not null"`
	Topic     string `xorm:"text"`
	IsForced  bool   `xorm:"bool not null"`
	IsDeleted bool   `xorm:"bool not null"`
	IsPublic  bool   `xorm:"bool not null"`
	IsHidden  bool   `xorm:"bool not null"`
	CreatorID string `xorm:"creator_id char(36) not null"`
	CreatedAt int    `xorm:"created not null"`
	UpdaterID string `xorm:"updater_id char(36) not null"`
	UpdatedAt int    `xorm:"updated not null"`
}

// TableName : テーブル名を指定するメソッド
func (channel *Channel) TableName() string {
	return "channels"
}

// Create : チャンネル作成を行うメソッド
func (channel *Channel) Create() error {
	// TODO:すでにIDが設定されていたらCreateじゃなくてUpdateなのでエラーを返す
	// TODO:英数字とアンダースコア・ハイフンだけを許容する
	if channel.Name == "" {
		return fmt.Errorf("Name is empty")
	}

	if channel.CreatorID == "" {
		return fmt.Errorf("CreatorId is empty")
	}
	channel.ID = CreateUUID()

	channel.UpdaterID = channel.CreatorID

	if _, err := db.Insert(channel); err != nil {
		return fmt.Errorf("Failed to create channel: %v", err)
	}
	return nil
}

// GetChannelByID : チャンネルIDによってチャンネルを取得
func GetChannelByID(userID, channelID string) (*Channel, error) {
	channel := new(Channel)
	channel.ID = channelID
	_, err := db.Get(channel)

	if err != nil {
		return nil, fmt.Errorf("Failed to get channel: %v", err)
	}

	return channel, nil
}

// ExistsChannel : 指定したチャンネルIDが存在するかどうかを確認する
func ExistsChannel(channelID string) (bool, error) {
	channel := Channel{ID: channelID}
	return db.Get(&channel)
}

// GetChannelList : userIDのユーザーから見えるチャンネルの一覧を取得する
func GetChannelList(userID string) ([]*Channel, error) {
	// TODO: 隠しチャンネルを表示するかどうかをクライアントと決める
	var channelList []*Channel
	err := db.Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("(is_public = true OR user_id = ?) AND is_deleted = false", userID).Find(&channelList)

	if err != nil {
		return nil, fmt.Errorf("Failed to find channels: %v", err)
	}
	return channelList, nil
}

// GetChildrenChannelIDList :  userIDのユーザーから見えるchannelIDの子チャンネル
func GetChildrenChannelIDList(userID, channelID string) ([]string, error) {
	var channelIDList []string
	// err := db.Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("is_public = true OR user_id = ?", userId).Cols("id").Find(&channelIdList)
	err := db.Table("channels").Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("(is_public = true OR user_id = ?) AND parent_id = ? AND is_deleted = false", userID, channelID).Cols("id").Find(&channelIDList)

	if err != nil {
		return nil, fmt.Errorf("Failed to find channels: %v", err)
	}
	return channelIDList, nil
}

// Update : チャンネルの情報の更新を行う
func (channel *Channel) Update() error {
	_, err := db.Id(channel.ID).UseBool().Update(channel)
	if err != nil {
		return fmt.Errorf("Failed to update channel: %v", err)
	}
	return nil
}
