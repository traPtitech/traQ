package model

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// MessageReport メッセージレポート構造体
type MessageReport struct {
	ID        uuid.UUID      `gorm:"type:char(36);not null;primaryKey"                   json:"id"`
	MessageID uuid.UUID      `gorm:"type:char(36);not null;uniqueIndex:message_reporter" json:"messageId"`
	Reporter  uuid.UUID      `gorm:"type:char(36);not null;uniqueIndex:message_reporter" json:"reporter"`
	Reason    string         `gorm:"type:TEXT COLLATE utf8mb4_bin NOT NULL"                json:"reason"`
	CreatedAt time.Time      `gorm:"precision:6;index"                                    json:"createdAt"`
	DeletedAt gorm.DeletedAt `gorm:"precision:6"                                          json:"-"`
}

// TableName MessageReport構造体のテーブル名
func (*MessageReport) TableName() string {
	return "message_reports"
}
