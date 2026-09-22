package gorm

import (
	"context"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/gormutil"
)

// CreateMessageReport implements MessageReportRepository interface.
func (repo *Repository) CreateMessageReport(ctx context.Context, messageID, reporterID uuid.UUID, reason string) error {
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
	if err := repo.db.WithContext(ctx).Create(r).Error; err != nil {
		if gormutil.IsMySQLDuplicatedRecordErr(err) {
			return repository.ErrAlreadyExists
		}
		return err
	}
	return nil
}

// GetMessageReports implements MessageReportRepository interface.
func (repo *Repository) GetMessageReports(ctx context.Context, offset, limit int) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	err = repo.db.WithContext(ctx).Scopes(gormutil.LimitAndOffset(limit, offset)).Order("created_at").Find(&arr).Error
	return arr, err
}

// GetMessageReportsByMessageID implements MessageReportRepository interface.
func (repo *Repository) GetMessageReportsByMessageID(ctx context.Context, messageID uuid.UUID) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	if messageID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.WithContext(ctx).Where(&model.MessageReport{MessageID: messageID}).Order("created_at").Find(&arr).Error
	return arr, err
}

// GetMessageReportsByReporterID implements MessageReportRepository interface.
func (repo *Repository) GetMessageReportsByReporterID(ctx context.Context, reporterID uuid.UUID) (arr []*model.MessageReport, err error) {
	arr = make([]*model.MessageReport, 0)
	if reporterID == uuid.Nil {
		return arr, nil
	}
	err = repo.db.WithContext(ctx).Where(&model.MessageReport{Reporter: reporterID}).Order("created_at").Find(&arr).Error
	return arr, err
}
