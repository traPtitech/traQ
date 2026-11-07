package gorm

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

// CreateImageEmbeddingJob implements ImageEmbeddingJobRepository interface.
func (repo *Repository) CreateImageEmbeddingJob(ctx context.Context, job *model.ImageEmbeddingJob) error {
	if job == nil || job.JobID == uuid.Nil {
		return repository.ErrNilID
	}
	return repo.db.WithContext(ctx).Create(job).Error
}

// GetImageEmbeddingJob implements ImageEmbeddingJobRepository interface.
func (repo *Repository) GetImageEmbeddingJob(ctx context.Context, jobID uuid.UUID) (*model.ImageEmbeddingJob, error) {
	if jobID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	job := &model.ImageEmbeddingJob{}
	if err := repo.db.WithContext(ctx).First(job, &model.ImageEmbeddingJob{JobID: jobID}).Error; err != nil {
		return nil, convertError(err)
	}
	return job, nil
}

// UpdateImageEmbeddingJobStatus implements ImageEmbeddingJobRepository interface.
func (repo *Repository) UpdateImageEmbeddingJobStatus(ctx context.Context, jobID uuid.UUID, status model.ImageEmbeddingJobStatus) error {
	if jobID == uuid.Nil {
		return repository.ErrNilID
	}
	return repo.db.WithContext(ctx).
		Model(&model.ImageEmbeddingJob{}).
		Where("job_id = ?", jobID).
		Update("status", status).
		Error
}

// GetUnfinishedImageEmbeddingJobs implements ImageEmbeddingJobRepository interface.
func (repo *Repository) GetUnfinishedImageEmbeddingJobs(ctx context.Context, limit int) ([]*model.ImageEmbeddingJob, error) {
	jobs := make([]*model.ImageEmbeddingJob, 0)
	tx := repo.db.WithContext(ctx).
		Where("status IN ?", []model.ImageEmbeddingJobStatus{
			model.ImageEmbeddingJobPending,
			model.ImageEmbeddingJobProcessing,
		}).
		Order("created_at ASC")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	if err := tx.Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// GetImageEmbeddingJobsByMessageIDs implements ImageEmbeddingJobRepository interface.
func (repo *Repository) GetImageEmbeddingJobsByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) ([]*model.ImageEmbeddingJob, error) {
	jobs := make([]*model.ImageEmbeddingJob, 0)
	if len(messageIDs) == 0 {
		return jobs, nil
	}
	if err := repo.db.WithContext(ctx).
		Where("message_id IN ?", messageIDs).
		Find(&jobs).
		Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// FailStaleImageEmbeddingJobs implements ImageEmbeddingJobRepository interface.
func (repo *Repository) FailStaleImageEmbeddingJobs(ctx context.Context, before time.Time) error {
	return repo.db.WithContext(ctx).
		Model(&model.ImageEmbeddingJob{}).
		Where("status IN ?", []model.ImageEmbeddingJobStatus{
			model.ImageEmbeddingJobPending,
			model.ImageEmbeddingJobProcessing,
		}).
		Where("created_at < ?", before).
		Update("status", model.ImageEmbeddingJobFailed).
		Error
}

// DeleteFinishedImageEmbeddingJobsBefore implements ImageEmbeddingJobRepository interface.
func (repo *Repository) DeleteFinishedImageEmbeddingJobsBefore(ctx context.Context, before time.Time) error {
	return repo.db.WithContext(ctx).
		Where("status IN ?", []model.ImageEmbeddingJobStatus{
			model.ImageEmbeddingJobCompleted,
			model.ImageEmbeddingJobFailed,
		}).
		Where("updated_at < ?", before).
		Delete(&model.ImageEmbeddingJob{}).
		Error
}

// DeleteAllImageEmbeddingJobs implements ImageEmbeddingJobRepository interface.
func (repo *Repository) DeleteAllImageEmbeddingJobs(ctx context.Context) error {
	return repo.db.WithContext(ctx).
		Where("1 = 1").
		Delete(&model.ImageEmbeddingJob{}).
		Error
}
