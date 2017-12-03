package model

import "fmt"

type Channels struct {
	Id        string `xorm:"char(36) primary_key"`
	Name      string `xorm:"varchar(20) not null primary_key"`
	ParentId  string `xorm:"char(36) not null"`
	CreatorId string `xorm:"char(36) not null"`
	Topic     string `xorm:"text"`
	IsForced  bool   `xorm:"boolean not null"`
	IsDeleted bool   `xorm:"boolean not null"`
	IsPublic  bool   `xorm:"boolean not null"`
	IsHidden  bool   `xorm:"boolean not null"`
	CreatedAt int    `xorm:"created not null"`
	UpdaterId string `xorm:"char(36) not null"`
	UpdatedAt int    `xorm:"updated not null"`
}

func (self *Channel) Create() error {
	if self.Name == "" {
		return fmt.Errorf("Name is empty")
	}

	if self.ParentId == "" {
		return fmt.Errorf("ParentId is empty")
	}

	if self.CreatorId == "" {
		return fmt.Errorf("CreatorId is empty")
	}

	self.UpdaterId = self.CreatorId

	if _, err := db.Insert(self); err != nil {
		return fmt.Errorf("Failed to create channel: %v", err)
	}
	return nil
}

func (self *Channel) Update() error {
	_, err := db.Id(self.Id).Update(self)
	if err != nil {
		return fmt.Errorf("Failed to update channel: %v", err)
	}
}
