package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/validator"
)

// CreateStamp スタンプを作成します
func (repo *RepositoryImpl) CreateStamp(name string, fileID, userID uuid.UUID) (s *model.Stamp, err error) {
	if fileID == uuid.Nil {
		return nil, ErrNilID
	}

	stamp := &model.Stamp{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		CreatorID: userID,
		FileID:    fileID,
	}
	if err := stamp.Validate(); err != nil {
		return nil, err
	}
	if err := repo.db.Create(stamp).Error; err != nil {
		if isMySQLDuplicatedRecordErr(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}
	repo.hub.Publish(hub.Message{
		Name: event.StampCreated,
		Fields: hub.Fields{
			"stamp":    stamp,
			"stamp_id": stamp.ID,
		},
	})
	return stamp, nil
}

// UpdateStamp スタンプの情報を更新します
func (repo *RepositoryImpl) UpdateStamp(id uuid.UUID, name string, fileID uuid.UUID) error {
	if id == uuid.Nil {
		return ErrNilID
	}

	data := map[string]string{}
	if len(name) > 0 {
		if err := validator.ValidateVar(name, "name"); err != nil {
			return err
		}
		data["name"] = name
	}
	if fileID != uuid.Nil {
		data["file_id"] = fileID.String()
	}
	if len(data) == 0 {
		return ErrInvalidArgs
	}

	result := repo.db.Model(&model.Stamp{}).Where(&model.Stamp{ID: id}).Updates(data)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.StampUpdated,
			Fields: hub.Fields{
				"stamp_id": id,
			},
		})
		return nil
	}
	return ErrNotFound
}

// GetStamp 指定したIDのスタンプを取得します
func (repo *RepositoryImpl) GetStamp(id uuid.UUID) (s *model.Stamp, err error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	s = &model.Stamp{}
	if err := repo.db.Where(&model.Stamp{ID: id}).Take(s).Error; err != nil {
		if gorm.IsRecordNotFoundError(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s, nil
}

// DeleteStamp 指定したIDのスタンプを削除します
func (repo *RepositoryImpl) DeleteStamp(id uuid.UUID) (err error) {
	if id == uuid.Nil {
		return ErrNilID
	}
	result := repo.db.Delete(&model.Stamp{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.StampDeleted,
			Fields: hub.Fields{
				"stamp_id": id,
			},
		})
		return nil
	}
	return ErrNotFound
}

// GetAllStamps 全てのスタンプを取得します
func (repo *RepositoryImpl) GetAllStamps() (stamps []*model.Stamp, err error) {
	stamps = make([]*model.Stamp, 0)
	err = repo.db.Find(&stamps).Error
	return stamps, err
}

// StampExists 指定したIDのスタンプが存在するかどうか
func (repo *RepositoryImpl) StampExists(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}
	c := 0
	err := repo.db.
		Model(&model.Stamp{}).
		Where(&model.Stamp{ID: id}).
		Limit(1).
		Count(&c).
		Error
	return c > 0, err
}

// IsStampNameDuplicate 指定した名前のスタンプが存在するかどうか
func (repo *RepositoryImpl) IsStampNameDuplicate(name string) (bool, error) {
	if len(name) == 0 {
		return false, nil
	}
	c := 0
	err := repo.db.
		Model(&model.Stamp{}).
		Where(&model.Stamp{Name: name}).
		Limit(1).
		Count(&c).
		Error
	return c > 0, err
}
