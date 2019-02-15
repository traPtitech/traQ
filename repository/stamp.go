package repository

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
)

// StampRepository スタンプリポジトリ
type StampRepository interface {
	CreateStamp(name string, fileID, userID uuid.UUID) (s *model.Stamp, err error)
	UpdateStamp(id uuid.UUID, name string, fileID uuid.UUID) error
	GetStamp(id uuid.UUID) (s *model.Stamp, err error)
	DeleteStamp(id uuid.UUID) (err error)
	GetAllStamps() (stamps []*model.Stamp, err error)
	StampExists(id uuid.UUID) (bool, error)
	IsStampNameDuplicate(name string) (bool, error)
}
