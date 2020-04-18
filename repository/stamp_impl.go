package repository

import (
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/validator"
)

// CreateStamp implements StampRepository interface.
func (repo *GormRepository) CreateStamp(args CreateStampArgs) (s *model.Stamp, err error) {
	stamp := &model.Stamp{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      args.Name,
		FileID:    args.FileID,
		CreatorID: args.CreatorID, // uuid.Nilを許容する
		IsUnicode: args.IsUnicode,
	}

	err = repo.db.Transaction(func(tx *gorm.DB) error {
		// 名前チェック
		if err := vd.Validate(stamp.Name, validator.StampNameRuleRequired...); err != nil {
			return ArgError("name", "Name must be 1-32 characters of a-zA-Z0-9_-")
		}
		// 名前重複チェック
		if exists, err := dbExists(tx, &model.Stamp{Name: stamp.Name}); err != nil {
			return err
		} else if exists {
			return ErrAlreadyExists
		}
		// ファイル存在チェック
		if stamp.FileID == uuid.Nil {
			return ArgError("fileID", "FileID's file is not found")
		}
		if exists, err := dbExists(tx, &model.File{ID: stamp.FileID}); err != nil {
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
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var s model.Stamp
		if err := tx.First(&s, &model.Stamp{ID: id}).Error; err != nil {
			return convertError(err)
		}

		if args.Name.Valid {
			if err := vd.Validate(args.Name.String, validator.StampNameRuleRequired...); err != nil {
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

// GetStampByName implements StampRepository interface.
func (repo *GormRepository) GetStampByName(name string) (s *model.Stamp, err error) {
	if len(name) == 0 {
		return nil, ErrNotFound
	}
	s = &model.Stamp{}
	if err := repo.db.Take(s, &model.Stamp{Name: name}).Error; err != nil {
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
func (repo *GormRepository) GetAllStamps(excludeUnicode bool) (stamps []*model.Stamp, err error) {
	stamps = make([]*model.Stamp, 0)
	tx := repo.db
	if excludeUnicode {
		tx = tx.Where("is_unicode = FALSE")
	}
	return stamps, tx.Find(&stamps).Error
}

// StampExists implements StampRepository interface.
func (repo *GormRepository) StampExists(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}
	return dbExists(repo.db, &model.Stamp{ID: id})
}

// ExistStamps implements StampPaletteRepository interface.
func (repo *GormRepository) ExistStamps(stampIDs []uuid.UUID) (err error) {
	var num int
	err = repo.db.
		Table("stamps").
		Where("id IN (?)", stampIDs).
		Count(&num).
		Error
	if err != nil {
		return err
	}
	if len(stampIDs) != num {
		err = ArgError("stamp", "stamp is not found")
	}
	return
}

// StampNameExists implements StampRepository interface.
func (repo *GormRepository) StampNameExists(name string) (bool, error) {
	if len(name) == 0 {
		return false, nil
	}
	return dbExists(repo.db, &model.Stamp{Name: name})
}

// GetUserStampHistory implements StampRepository interface.
func (repo *GormRepository) GetUserStampHistory(userID uuid.UUID, limit int) (h []*UserStampHistory, err error) {
	h = make([]*UserStampHistory, 0)
	if userID == uuid.Nil {
		return
	}
	err = repo.db.
		Table("messages_stamps").
		Where("user_id = ?", userID).
		Group("stamp_id").
		Select("stamp_id, max(updated_at) AS datetime").
		Order("datetime DESC").
		Scopes(limitAndOffset(limit, 0)).
		Scan(&h).
		Error
	return
}
