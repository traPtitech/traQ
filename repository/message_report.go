package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// MessageReportRepository メッセージ通報リポジトリ
type MessageReportRepository interface {
	CreateMessageReport(messageID, reporterID uuid.UUID, reason string) error
	GetMessageReports(offset, limit int) ([]*model.MessageReport, error)
	GetMessageReportsByMessageID(messageID uuid.UUID) ([]*model.MessageReport, error)
	GetMessageReportsByReporterID(reporterID uuid.UUID) ([]*model.MessageReport, error)
}
