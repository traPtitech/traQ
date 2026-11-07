package model

import (
	"time"

	"github.com/gofrs/uuid"
)

// ImageEmbeddingJobStatus 画像埋め込みジョブの状態
type ImageEmbeddingJobStatus string

const (
	// ImageEmbeddingJobPending 処理待ち
	ImageEmbeddingJobPending ImageEmbeddingJobStatus = "pending"
	// ImageEmbeddingJobProcessing 処理中
	ImageEmbeddingJobProcessing ImageEmbeddingJobStatus = "processing"
	// ImageEmbeddingJobCompleted 完了
	ImageEmbeddingJobCompleted ImageEmbeddingJobStatus = "completed"
	// ImageEmbeddingJobFailed 失敗
	ImageEmbeddingJobFailed ImageEmbeddingJobStatus = "failed"
)

// IsFinished ジョブが終了状態かどうかを返します
func (s ImageEmbeddingJobStatus) IsFinished() bool {
	return s == ImageEmbeddingJobCompleted || s == ImageEmbeddingJobFailed
}

// ImageEmbeddingJob embedding-serverの画像埋め込みジョブ
type ImageEmbeddingJob struct {
	JobID     uuid.UUID               `gorm:"type:char(36);not null;primaryKey"`
	MessageID uuid.UUID               `gorm:"type:char(36);not null;index"`
	FileID    uuid.UUID               `gorm:"type:char(36);not null;index"`
	Status    ImageEmbeddingJobStatus `gorm:"type:varchar(20);not null;index"`
	CreatedAt time.Time               `gorm:"precision:6"`
	UpdatedAt time.Time               `gorm:"precision:6"`
}

// TableName テーブル名
func (*ImageEmbeddingJob) TableName() string {
	return "image_embedding_jobs"
}
