package gorm

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/gormutil"
)

// CreateMessageReport implements MessageReportRepository interface.
func (repo *Repository) CreateMessageReport(messageID, reporterID uuid.UUID, reason string) error {
	// nil check
	if messageID == uuid.Nil || reporterID == uuid.Nil {
		return repository.ErrNilID
	}

	// make report
	r := &model.MessageReport{
		ID:        uuid.Must(uuid.NewV7()),
		MessageID: messageID,
		Reporter:  reporterID,
		Reason:    reason,
	}
	if err := repo.db.Create(r).Error; err != nil {
		if gormutil.IsMySQLDuplicatedRecordErr(err) {
			return repository.ErrAlreadyExists
		}
		return err
	}
	return nil
}

// GetMessageReports implements MessageReportRepository interface.
func (repo *Repository) GetMessageReports(offset, limit int) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	err = repo.db.Scopes(gormutil.LimitAndOffset(limit, offset)).Order("created_at").Find(&arr).Error
	return arr, err
}

// GetMessageReportsByMessageID implements MessageReportRepository interface.
func (repo *Repository) GetMessageReportsByMessageID(messageID uuid.UUID) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	if messageID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.Where(&model.MessageReport{MessageID: messageID}).Order("created_at").Find(&arr).Error
	return arr, err
}

// GetMessageReportsByReporterID implements MessageReportRepository interface.
func (repo *Repository) GetMessageReportsByReporterID(reporterID uuid.UUID) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	if reporterID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.Where(&model.MessageReport{Reporter: reporterID}).Order("created_at").Find(&arr).Error
	return arr, err
}
