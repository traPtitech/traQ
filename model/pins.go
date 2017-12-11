package model

import (
  "fmt"
)

type Pins struct {
  ChannelId string 'xorm:"char(36) not null"'
  MessageId string 'xorm:"char(36) not null"'
  UserId    string 'xorm:"char(36) not null"'
  CreateAt  string 'xorm:"created not null"'
}

func (pins *Pins) Create() error {
  if pins.UserId == "" {
    return fmt.Errorf("UserId is empty")
  }
  if pins.ChannelId == "" {
    return fmt.Errorf("ChannelId is empty")
  }
  if pins.MessageId == "" {
    return fmt.Errorf("MessageId is empty")
  }

  if _, err := db.Insert(pins); err != nil {
    return fmt.Errorf("Failed to create pin object: %v", err)
  }

  return nil
}

}

func (pins *Pins) GetPinnedMessage(channelId string) (*pinnedMessage, error){
  pinnedMessage := &Pins{}
  pinnedMessage.ChannelId = channelId
  _, err := db.Get("pinnedMessage")

  if err != nil {
    return nil, fmt.Errorf("Failed to find pin: %v", err)
  }
  return pinnedMessage, nil
}
