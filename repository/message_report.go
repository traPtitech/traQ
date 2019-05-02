package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// MessageReportRepository メッセージ通報リポジトリ
type MessageReportRepository interface {
	// CreateMessageReport 指定したユーザーによる指定したメッセージの通報を登録します
	//
	// 成功した場合、nilを返します。
	// 既に通報がされていた場合、ErrAlreadyExistsを返します。
	// 引数にuuid.Nilを指定するとErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	CreateMessageReport(messageID, reporterID uuid.UUID, reason string) error
	// GetMessageReports メッセージ通報を通報日時の昇順で取得します
	//
	// 成功した場合、メッセージ通報の配列とnilを返します。負のoffset, limitは無視されます。
	// DBによるエラーを返すことがあります。
	GetMessageReports(offset, limit int) ([]*model.MessageReport, error)
	// GetMessageReportsByMessageID 指定したメッセージのメッセージ通報を全て取得します
	//
	// 成功した場合、メッセージ通報の配列とnilを返します。
	// 存在しないメッセージを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetMessageReportsByMessageID(messageID uuid.UUID) ([]*model.MessageReport, error)
	// GetMessageReportsByReporterID 指定したユーザーによるメッセージ通報を全て取得します
	//
	// 成功した場合、メッセージ通報の配列とnilを返します。
	// 存在しないユーザーを指定した場合は空配列とnilを返します。
	// DBによるエラーを返すことがあります。
	GetMessageReportsByReporterID(reporterID uuid.UUID) ([]*model.MessageReport, error)
}
