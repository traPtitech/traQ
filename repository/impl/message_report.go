package impl

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

// CreateMessageReport 指定したメッセージの通報を登録します
func (repo *RepositoryImpl) CreateMessageReport(messageID, reporterID uuid.UUID, reason string) error {
	// nil check
	if messageID == uuid.Nil || reporterID == uuid.Nil {
		return repository.ErrNilID
	}

	// make report
	r := &model.MessageReport{
		ID:        uuid.NewV4(),
		MessageID: messageID,
		Reporter:  reporterID,
		Reason:    reason,
	}
	return repo.db.Create(r).Error
}

// GetMessageReports メッセージ通報を通報日時の昇順で取得します
func (repo *RepositoryImpl) GetMessageReports(offset, limit int) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	err = repo.db.Scopes(limitAndOffset(limit, offset)).Order("created_at").Find(&arr).Error
	return arr, err
}

// GetMessageReportsByMessageID メッセージ通報を取得します
func (repo *RepositoryImpl) GetMessageReportsByMessageID(messageID uuid.UUID) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	if messageID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.Where(&model.MessageReport{MessageID: messageID}).Order("created_at").Find(&arr).Error
	return arr, err
}

// GetMessageReportsByReporterID メッセージ通報を取得します
func (repo *RepositoryImpl) GetMessageReportsByReporterID(reporterID uuid.UUID) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	if reporterID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.Where(&model.MessageReport{Reporter: reporterID}).Order("created_at").Find(&arr).Error
	return arr, err
}
