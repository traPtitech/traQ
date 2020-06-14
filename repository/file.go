package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
)

// FilesQuery GetFiles用クエリ
type FilesQuery struct {
	UploaderID optional.UUID
	ChannelID  optional.UUID
	Since      optional.Time
	Until      optional.Time
	Inclusive  bool
	Limit      int
	Offset     int
	Asc        bool
	Type       model.FileType
}

// FileRepository ファイルリポジトリ
type FileRepository interface {
	GetFileMetas(q FilesQuery) (result []*model.FileMeta, more bool, err error)
	GetFileMeta(fileID uuid.UUID) (*model.FileMeta, error)
	SaveFileMeta(meta *model.FileMeta, acl []*model.FileACLEntry) error
	DeleteFileMeta(fileID uuid.UUID) error
	IsFileAccessible(fileID, userID uuid.UUID) (bool, error)
}
