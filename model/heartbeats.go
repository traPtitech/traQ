package model

import (
	"github.com/satori/go.uuid"
	"time"
)

// UserStatus userの状態
type UserStatus struct {
	UserID   uuid.UUID `json:"userId"`
	Status   string    `json:"status"`
	LastTime time.Time `json:"-"`
}

// HeartbeatStatus Heartbeatの状態
type HeartbeatStatus struct {
	UserStatuses []*UserStatus `json:"userStatuses"`
	ChannelID    uuid.UUID     `json:"channelId"`
}
