package gorm

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

func TestRepositoryImpl_ImageEmbeddingJob(t *testing.T) {
	t.Parallel()
	repo, assert, require := setup(t, common)
	ctx := context.Background()

	jobID := uuid.Must(uuid.NewV4())
	messageID := uuid.Must(uuid.NewV4())
	fileID := uuid.Must(uuid.NewV4())

	assert.Error(repo.CreateImageEmbeddingJob(ctx, nil))
	assert.Error(repo.CreateImageEmbeddingJob(ctx, &model.ImageEmbeddingJob{MessageID: messageID, FileID: fileID}))
	_, err := repo.GetImageEmbeddingJob(ctx, uuid.Nil)
	assert.Error(err)
	assert.Error(repo.UpdateImageEmbeddingJobStatus(ctx, uuid.Nil, model.ImageEmbeddingJobFailed))

	job := &model.ImageEmbeddingJob{
		JobID:     jobID,
		MessageID: messageID,
		FileID:    fileID,
		Status:    model.ImageEmbeddingJobPending,
		CreatedAt: time.Now().Add(-2 * time.Hour),
		UpdatedAt: time.Now().Add(-2 * time.Hour),
	}
	require.NoError(repo.CreateImageEmbeddingJob(ctx, job))

	got, err := repo.GetImageEmbeddingJob(ctx, jobID)
	require.NoError(err)
	assert.Equal(model.ImageEmbeddingJobPending, got.Status)

	require.NoError(repo.UpdateImageEmbeddingJobStatus(ctx, jobID, model.ImageEmbeddingJobProcessing))
	got, err = repo.GetImageEmbeddingJob(ctx, jobID)
	require.NoError(err)
	assert.Equal(model.ImageEmbeddingJobProcessing, got.Status)

	jobs, err := repo.GetUnfinishedImageEmbeddingJobs(ctx, 1000)
	require.NoError(err)
	found := false
	for _, j := range jobs {
		if j.JobID == jobID {
			found = true
			break
		}
	}
	assert.True(found)

	byMessage, err := repo.GetImageEmbeddingJobsByMessageIDs(ctx, []uuid.UUID{messageID})
	require.NoError(err)
	require.Len(byMessage, 1)
	assert.Equal(jobID, byMessage[0].JobID)

	require.NoError(repo.FailStaleImageEmbeddingJobs(ctx, time.Now().Add(-time.Hour)))
	got, err = repo.GetImageEmbeddingJob(ctx, jobID)
	require.NoError(err)
	assert.Equal(model.ImageEmbeddingJobFailed, got.Status)

	require.NoError(repo.DeleteFinishedImageEmbeddingJobsBefore(ctx, time.Now().Add(time.Hour)))
	_, err = repo.GetImageEmbeddingJob(ctx, jobID)
	assert.Error(err)

	job2 := &model.ImageEmbeddingJob{
		JobID:     uuid.Must(uuid.NewV4()),
		MessageID: uuid.Must(uuid.NewV4()),
		FileID:    uuid.Must(uuid.NewV4()),
		Status:    model.ImageEmbeddingJobCompleted,
	}
	require.NoError(repo.CreateImageEmbeddingJob(ctx, job2))
	require.NoError(repo.DeleteAllImageEmbeddingJobs(ctx))
	_, err = repo.GetImageEmbeddingJob(ctx, job2.JobID)
	assert.Error(err)
}
