package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"io"
)

// FileRepository ファイルリポジトリ
type FileRepository interface {
	OpenFile(fileID uuid.UUID) (*model.File, io.ReadCloser, error)
	OpenThumbnailFile(fileID uuid.UUID) (*model.File, io.ReadCloser, error)
	GetFileMeta(fileID uuid.UUID) (*model.File, error)
	DeleteFile(fileID uuid.UUID) error
	GenerateIconFile(salt string) (uuid.UUID, error)
	SaveFile(name string, src io.Reader, size int64, mime string, fType string, creatorID uuid.UUID) (*model.File, error)
	SaveFileWithACL(name string, src io.Reader, size int64, mime string, fType string, creatorID uuid.UUID, read ACL) (*model.File, error)
	RegenerateThumbnail(fileID uuid.UUID) (bool, error)
	IsFileAccessible(fileID, userID uuid.UUID) (bool, error)
}

// ACL アクセスコントロールリスト
type ACL map[uuid.UUID]bool
