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

// CreateStampPalette implements StampPaletteRepository interface.
func (repo *GormRepository) CreateStampPalette(name, description string, stamps model.UUIDs, userID uuid.UUID) (sp *model.StampPalette, err error) {
	if userID == uuid.Nil {
		return nil, ErrNilID
	}
	stampPalette := &model.StampPalette{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		Description: description,
		Stamps: stamps,
		CreatorID: userID,
	}

	err = repo.db.Transaction(func(tx *gorm.DB) error {
		// 名前チェック
		if err := validation.Validate(name, validator.StampPaletteNameRuleRequired...); err != nil {
			return ArgError("name", "Name must be 1-30")
		}
		// 説明チェック
		if err = validation.Validate(description, validator.StampPaletteDescriptionRuleRequired...); err != nil {
			return ArgError("description", "Description must be 0-1000")
		}
		// スタンプ存在チェック
		// dbExistの重さが分らないですが、クエリ数的には全件に対してやると、ここめちゃくちゃ重そう
		for _, stamp := range stamps {
			if exists, err := repo.StampExists(stamp); err != nil {
				return err
			} else if !exists {
				return ArgError("stamp", "stamp is not found")
			}
		}

		return tx.Create(stampPalette).Error
	})
	if err != nil {
		return nil, err
	}

	repo.hub.Publish(hub.Message{
		Name: event.StampCreated,
		Fields: hub.Fields{
			"stamp_palette_id": stampPalette.ID,
			"stamp_palette": stampPalette,
		},
	})
	return stampPalette, nil
}

// GetStampPalette implements StampPaletteRepository interface.
func (repo *GormRepository) GetStampPalette(id uuid.UUID) (sp *model.StampPalette, err error) {
	if id == uuid.Nil {
		return nil, ErrNilID
	}
	sp = &model.StampPalette{}
	if err := repo.db.Take(sp, &model.StampPalette{ID: id}).Error; err != nil {
		return nil, convertError(err)
	}
	return sp, nil
}

// DeleteStampPalette implements StampPaletteRepository interface.
func (repo *GormRepository) DeleteStampPalette(id uuid.UUID) (err error) {
	if id == uuid.Nil {
		return ErrNilID
	}
	result := repo.db.Delete(&model.StampPalette{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.StampPaletteDeleted,
			Fields: hub.Fields{
				"stamp_palette_id": id,
			},
		})
		return nil
	}
	return ErrNotFound
}

// GetAllStamps implements StampPaletteRepository interface.
func (repo *GormRepository) GetStampPalettes(userID uuid.UUID) (sps []*model.StampPalette, err error) {
	sps = make([]*model.StampPalette, 0)
	tx := repo.db
	return sps, tx.Find(&sps).Error
}
