package model


type Messages struct {
	Id string `xorm:char(36) pk`
	UserId string `xorm:char(36) not null`
	ChannelId string `xorm:char(36)`
	text string `xorm:text not null`
	IsShared bool `xorm:bool not null`
	IsDeleted bool `xorm:bool not null`
	CreatedAt string `xorm:created not null`
	UpdaterId string `xorm:char(36) not null`
	UpdatedAt string `xorm:updated not null`
}

func (self *Messages) Create() error {
	return nil
} 