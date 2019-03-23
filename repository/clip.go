package repository

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
)

// ClipRepository クリップリポジトリ
type ClipRepository interface {
	GetClipFolder(id uuid.UUID) (*model.ClipFolder, error)
	GetClipFolders(userID uuid.UUID) ([]*model.ClipFolder, error)
	CreateClipFolder(userID uuid.UUID, name string) (*model.ClipFolder, error)
	UpdateClipFolderName(id uuid.UUID, name string) error
	DeleteClipFolder(id uuid.UUID) error
	GetClipMessage(id uuid.UUID) (*model.Clip, error)
	GetClipMessages(folderID uuid.UUID) ([]*model.Clip, error)
	GetClipMessagesByUser(userID uuid.UUID) ([]*model.Clip, error)
	CreateClip(messageID, folderID, userID uuid.UUID) (*model.Clip, error)
	ChangeClipFolder(clipID, folderID uuid.UUID) error
	DeleteClip(id uuid.UUID) error
}
