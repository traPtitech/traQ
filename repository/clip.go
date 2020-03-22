package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// ClipFolderMessageQuery クリップフォルダー内のメッセージ取得用クエリ
type ClipFolderMessageQuery struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
	Asc    bool
}

type CliplRepository interface {
	CreateClipFolder(userID uuid.UUID, name string, description string) (*model.ClipFolder, error)
	UpdateClipFolder(folderID uuid.UUID, name string, description string) error
	DeleteClipFolder(folderID uuid.UUID) error
	DeleteClipFolderMessage(folderID, messageID uuid.UUID) error
	AddClipFolderMessage(folderID, messageID uuid.UUID) (*model.Message, error)
	GetClipFoldersByUserID(userID uuid.UUID) ([]*model.ClipFolder, error)
	GetClipFolder(folderID uuid.UUID) (*model.ClipFolder, error)
	GetClipFolderMessages(folderID uuid.UUID, query ClipFolderMessageQuery) (messages []*model.Message, more bool, err error)
}
