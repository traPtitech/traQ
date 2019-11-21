package repository

import (
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/validator"
)

// CreateStamp implements StampRepository interface.
func (repo *GormRepository) CreateStamp(name string, fileID, userID uuid.UUID) (s *model.Stamp, err error) {
	stamp := &model.Stamp{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		FileID:    fileID,
		CreatorID: userID, // uuid.Nilを許容する
	}

	err = repo.transact(func(tx *gorm.DB) error {
		// 名前チェック
		if err := validation.Validate(name, validator.StampNameRuleRequired...); err != nil {
			return ArgError("name", "Name must be 1-32 characters of a-zA-Z0-9_-")
		}
		// 名前重複チェック
		if exists, err := dbExists(tx, &model.Stamp{Name: name}); err != nil {
			return err
		} else if exists {
			return ErrAlreadyExists
		}
		// ファイル存在チェック
		if fileID == uuid.Nil {
			return ArgError("fileID", "FileID's file is not found")
		}
		if exists, err := dbExists(tx, &model.File{ID: fileID}); err != nil {
			return err
		} else if !exists {
			return ArgError("fileID", "fileID's file is not found")
		}

		return tx.Create(stamp).Error
	})
	if err != nil {
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

// UpdateStamp implements StampRepository interface.
func (repo *GormRepository) UpdateStamp(id uuid.UUID, args UpdateStampArgs) error {
	if id == uuid.Nil {
		return ErrNilID
	}
	changes := map[string]interface{}{}
	err := repo.transact(func(tx *gorm.DB) error {
		var s model.Stamp
		if err := tx.First(&s, &model.Stamp{ID: id}).Error; err != nil {
			return convertError(err)
		}

		if args.Name.Valid {
			if err := validation.Validate(args.Name.String, validator.StampNameRuleRequired...); err != nil {
				return ArgError("args.Name", "Name must be 1-32 characters of a-zA-Z0-9_-")
			}

			// 重複チェック
			if exists, err := dbExists(tx, &model.Stamp{Name: args.Name.String}); err != nil {
				return err
			} else if exists {
				return ErrAlreadyExists
			}
			changes["name"] = args.Name.String
		}
		if args.FileID.Valid {
			// 存在チェック
			if args.FileID.UUID == uuid.Nil {
				return ArgError("args.FileID", "FileID's file is not found")
			}
			if exists, err := dbExists(tx, &model.File{ID: args.FileID.UUID}); err != nil {
				return err
			} else if !exists {
				return ArgError("args.FileID", "FileID's file is not found")
			}
			changes["file_id"] = args.FileID.UUID
		}
		if args.CreatorID.Valid {
			// uuid.Nilを許容する
			changes["creator_id"] = args.CreatorID.UUID
		}

		if len(changes) > 0 {
			return tx.Model(&s).Updates(changes).Error
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(changes) > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.StampUpdated,
			Fields: hub.Fields{
				"stamp_id": id,
			},
		})
	}
	return nil
}

// GetStamp implements StampRepository interface.
func (repo *GormRepository) GetStamp(id uuid.UUID) (s *model.Stamp, err error) {
	if id == uuid.Nil {
		return nil, ErrNotFound
	}
	s = &model.Stamp{}
	if err := repo.db.Take(s, &model.Stamp{ID: id}).Error; err != nil {
		return nil, convertError(err)
	}
	return s, nil
}

// DeleteStamp implements StampRepository interface.
func (repo *GormRepository) DeleteStamp(id uuid.UUID) (err error) {
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

// GetAllStamps implements StampRepository interface.
func (repo *GormRepository) GetAllStamps() (stamps []*model.Stamp, err error) {
	stamps = make([]*model.Stamp, 0)
	err = repo.db.Find(&stamps).Error
	return stamps, err
}

// StampExists implements StampRepository interface.
func (repo *GormRepository) StampExists(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}
	return dbExists(repo.db, &model.Stamp{ID: id})
}

// StampNameExists implements StampRepository interface.
func (repo *GormRepository) StampNameExists(name string) (bool, error) {
	if len(name) == 0 {
		return false, nil
	}
	return dbExists(repo.db, &model.Stamp{Name: name})
}
