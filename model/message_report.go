package model

import (
	"github.com/satori/go.uuid"
	"time"
)

// MessageReport メッセージレポート構造体
type MessageReport struct {
	ID        string     `gorm:"type:char(36);primary_key"                   json:"id"`
	MessageID string     `gorm:"type:char(36);unique_index:message_reporter" json:"messageId"`
	Reporter  string     `gorm:"type:char(36);unique_index:message_reporter" json:"reporter"`
	Reason    string     `gorm:"type:text"                                   json:"reason"`
	CreatedAt time.Time  `gorm:"precision:6;index"                           json:"createdAt"`
	DeletedAt *time.Time `gorm:"precision:6"                                 json:"-"`
}

// TableName MessageReport構造体のテーブル名
func (*MessageReport) TableName() string {
	return "message_reports"
}

// CreateMessageReport 指定したメッセージを通報します
func CreateMessageReport(messageID, reporterID uuid.UUID, reason string) error {
	r := &MessageReport{
		ID:        CreateUUID(),
		MessageID: messageID.String(),
		Reporter:  reporterID.String(),
		Reason:    reason,
	}
	return db.Create(r).Error
}

// GetMessageReports メッセージ通報を通報日時の昇順で取得します
func GetMessageReports(offset, limit int) (arr []*MessageReport, err error) {
	err = db.Order("created_at").Limit(limit).Offset(offset).Find(&arr).Error
	return
}

// GetMessageReportsByMessageID メッセージ通報を取得します
func GetMessageReportsByMessageID(messageID uuid.UUID) (arr []*MessageReport, err error) {
	err = db.Where(MessageReport{MessageID: messageID.String()}).Find(&arr).Order("created_at").Error
	return
}

// GetMessageReportsByReporterID メッセージ通報を取得します
func GetMessageReportsByReporterID(reporterID uuid.UUID) (arr []*MessageReport, err error) {
	err = db.Where(MessageReport{Reporter: reporterID.String()}).Find(&arr).Order("created_at").Error
	return
}
