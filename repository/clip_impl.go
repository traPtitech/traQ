package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/model"
)

func (repo *GormRepository) CreateClipFolder(userID uuid.UUID, name string, description string) (*model.ClipFolder, error) {
	uid := uuid.Must(uuid.NewV4())
	//description のバリデーションする
	clipFolder := &model.ClipFolder{
		ID:          uid,
		Description: description,
		Name:        name,
		OwnerID:     userID,
	}
	if err := repo.db.Create(clipFolder).Error; err != nil {
		return nil, err
	}
	return clipFolder, nil
}
