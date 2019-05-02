package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// CreateMessageReport 指定したメッセージの通報を登録します
func (repo *GormRepository) CreateMessageReport(messageID, reporterID uuid.UUID, reason string) error {
	// nil check
	if messageID == uuid.Nil || reporterID == uuid.Nil {
		return ErrNilID
	}

	// make report
	r := &model.MessageReport{
		ID:        uuid.Must(uuid.NewV4()),
		MessageID: messageID,
		Reporter:  reporterID,
		Reason:    reason,
	}
	return repo.db.Create(r).Error
}

// GetMessageReports メッセージ通報を通報日時の昇順で取得します
func (repo *GormRepository) GetMessageReports(offset, limit int) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	err = repo.db.Scopes(limitAndOffset(limit, offset)).Order("created_at").Find(&arr).Error
	return arr, err
}

// GetMessageReportsByMessageID メッセージ通報を取得します
func (repo *GormRepository) GetMessageReportsByMessageID(messageID uuid.UUID) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	if messageID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.Where(&model.MessageReport{MessageID: messageID}).Order("created_at").Find(&arr).Error
	return arr, err
}

// GetMessageReportsByReporterID メッセージ通報を取得します
func (repo *GormRepository) GetMessageReportsByReporterID(reporterID uuid.UUID) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	if reporterID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.Where(&model.MessageReport{Reporter: reporterID}).Order("created_at").Find(&arr).Error
	return arr, err
}
