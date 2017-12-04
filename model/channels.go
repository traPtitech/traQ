package model

import "fmt"

type Channels struct {
	Id        string `xorm:"char(36) pk"`
	Name      string `xorm:"varchar(20) not null"`
	ParentId  string `xorm:"char(36) not null"`
	Topic     string `xorm:"text"`
	IsForced  bool   `xorm:"bool not null"`
	IsDeleted bool   `xorm:"bool not null"`
	IsPublic  bool   `xorm:"bool not null"`
	IsHidden  bool   `xorm:"bool not null"`
	CreatorId string `xorm:"char(36) not null"`
	CreatedAt int    `xorm:"created not null"`
	UpdaterId string `xorm:"char(36) not null"`
	UpdatedAt int    `xorm:"updated not null"`
}

func (self *Channels) Create() error {
	if self.Name == "" {
		return fmt.Errorf("Name is empty")
	}

	if self.CreatorId == "" {
		return fmt.Errorf("CreatorId is empty")
	}
	self.Id = CreateUUID()

	self.UpdaterId = self.CreatorId

	if _, err := db.Insert(self); err != nil {
		return fmt.Errorf("Failed to create channel: %v", err)
	}
	return nil
}

func GetChannelById(userId, channelId string) (*Channels, error) {
	channel := new(Channels)
	channel.Id = channelId
	_, err := db.Get(channel)

	if err != nil {
		return nil, fmt.Errorf("Failed to get channel: %v", err)
	}

	return channel, nil
}

func GetChannelList(userId string) ([]*Channels, error) {
	var channelList []*Channels
	err := db.Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("is_public = true OR user_id = ?", userId).Find(&channelList)

	if err != nil {
		return nil, fmt.Errorf("Failed to find channels: %v", err)
	}
	return channelList, nil
}

func GetChildrenChannelIdList(userId, channelId string) ([]string, error) {
	var channelIdList []string
	// err := db.Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("is_public = true OR user_id = ?", userId).Cols("id").Find(&channelIdList)
	err := db.Table("channels").Join("LEFT", "users_private_channels", "users_private_channels.channel_id = channels.id").Where("(is_public = true OR user_id = ?) AND parent_id = ?", userId, channelId).Cols("id").Find(&channelIdList)

	if err != nil {
		return nil, fmt.Errorf("Failed to find channels: %v", err)
	}
	return channelIdList, nil
}

func (self *Channels) Update() error {
	_, err := db.Id(self.Id).Update(self)
	if err != nil {
		return fmt.Errorf("Failed to update channel: %v", err)
	}
	return nil
}
