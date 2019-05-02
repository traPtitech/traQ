package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// CreateMessageReport implements MessageReportRepository interface.
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
	if err := repo.db.Create(r).Error; err != nil {
		if isMySQLDuplicatedRecordErr(err) {
			return ErrAlreadyExists
		}
		return err
	}
	return nil
}

// GetMessageReports implements MessageReportRepository interface.
func (repo *GormRepository) GetMessageReports(offset, limit int) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	err = repo.db.Scopes(limitAndOffset(limit, offset)).Order("created_at").Find(&arr).Error
	return arr, err
}

// GetMessageReportsByMessageID implements MessageReportRepository interface.
func (repo *GormRepository) GetMessageReportsByMessageID(messageID uuid.UUID) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	if messageID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.Where(&model.MessageReport{MessageID: messageID}).Order("created_at").Find(&arr).Error
	return arr, err
}

// GetMessageReportsByReporterID implements MessageReportRepository interface.
func (repo *GormRepository) GetMessageReportsByReporterID(reporterID uuid.UUID) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	if reporterID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.Where(&model.MessageReport{Reporter: reporterID}).Order("created_at").Find(&arr).Error
	return arr, err
}
