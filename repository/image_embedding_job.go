//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package repository

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// ImageEmbeddingJobRepository 画像埋め込みジョブリポジトリ
type ImageEmbeddingJobRepository interface {
	// CreateImageEmbeddingJob 画像埋め込みジョブを作成します
	//
	// jobがnil、またはJobIDがuuid.Nilの場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	CreateImageEmbeddingJob(ctx context.Context, job *model.ImageEmbeddingJob) error

	// GetImageEmbeddingJob 指定した画像埋め込みジョブを取得します
	//
	// 存在しなかった場合、ErrNotFoundを返します。
	// DBによるエラーを返すことがあります。
	GetImageEmbeddingJob(ctx context.Context, jobID uuid.UUID) (*model.ImageEmbeddingJob, error)

	// UpdateImageEmbeddingJobStatus 指定した画像埋め込みジョブの状態を更新します
	//
	// jobIDにuuid.Nilを指定した場合、ErrNilIDを返します。
	// DBによるエラーを返すことがあります。
	UpdateImageEmbeddingJobStatus(ctx context.Context, jobID uuid.UUID, status model.ImageEmbeddingJobStatus) error

	// GetUnfinishedImageEmbeddingJobs 終了していない画像埋め込みジョブを取得します
	//
	// 作成日時が古い順に最大limit件返します。limitが0以下の場合は無視します。
	// DBによるエラーを返すことがあります。
	GetUnfinishedImageEmbeddingJobs(ctx context.Context, limit int) ([]*model.ImageEmbeddingJob, error)

	// GetImageEmbeddingJobsByMessageIDs 指定したメッセージの画像埋め込みジョブを取得します
	//
	// DBによるエラーを返すことがあります。
	GetImageEmbeddingJobsByMessageIDs(ctx context.Context, messageIDs []uuid.UUID) ([]*model.ImageEmbeddingJob, error)

	// FailStaleImageEmbeddingJobs 指定した時刻より前に作成され、終了していないジョブを失敗にします
	//
	// DBによるエラーを返すことがあります。
	FailStaleImageEmbeddingJobs(ctx context.Context, before time.Time) error

	// DeleteFinishedImageEmbeddingJobsBefore 指定した時刻より前に更新され、終了したジョブを削除します
	//
	// DBによるエラーを返すことがあります。
	DeleteFinishedImageEmbeddingJobsBefore(ctx context.Context, before time.Time) error

	// DeleteAllImageEmbeddingJobs 全ての画像埋め込みジョブを削除します
	//
	// DBによるエラーを返すことがあります。
	DeleteAllImageEmbeddingJobs(ctx context.Context) error
}
