package model

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"time"
)

// MessageReport メッセージレポート構造体
type MessageReport struct {
	ID        string    `xorm:"char(36) pk"                                validate:"uuid,required"`
	MessageID string    `xorm:"char(36) not null unique(message_reporter)" validate:"uuid,required"`
	Reporter  string    `xorm:"char(36) not null unique(message_reporter)" validate:"uuid,required"`
	Reason    string    `xorm:"varchar(100) not null"                      validate:"max=100,required"`
	CreatedAt time.Time `xorm:"created"`
}

// TableName MessageReport構造体のテーブル名
func (*MessageReport) TableName() string {
	return "message_reports"
}

// Validate 構造体を検証します
func (m *MessageReport) Validate() error {
	return validator.ValidateStruct(m)
}

// CreateMessageReport 指定したメッセージを通報します
func CreateMessageReport(messageID, reporterID uuid.UUID, reason string) error {
	r := &MessageReport{
		ID:        CreateUUID(),
		MessageID: messageID.String(),
		Reporter:  reporterID.String(),
		Reason:    reason,
	}
	if err := r.Validate(); err != nil {
		return err
	}

	if _, err := db.InsertOne(r); err != nil {
		return err
	}
	return nil
}

// GetMessageReportsByMessageID メッセージ通報を取得します
func GetMessageReportsByMessageID(messageID uuid.UUID) (arr []*MessageReport, err error) {
	err = db.Where("message_id = ?", messageID.String()).Find(&arr)
	return
}
