package gorm

import (
	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/validator"
)

// CreateStampPalette implements StampPaletteRepository interface.
func (repo *Repository) CreateStampPalette(name, description string, stamps model.UUIDs, userID uuid.UUID) (sp *model.StampPalette, err error) {
	if userID == uuid.Nil {
		return nil, repository.ErrNilID
	}
	stampPalette := &model.StampPalette{
		ID:          uuid.Must(uuid.NewV7()),
		Name:        name,
		Description: description,
		Stamps:      stamps,
		CreatorID:   userID,
	}

	err = repo.db.Transaction(func(tx *gorm.DB) error {
		// 名前チェック
		if err := vd.Validate(name, validator.StampPaletteNameRuleRequired...); err != nil {
			return repository.ArgError("name", "Name must be 1-30")
		}
		// 説明チェック
		if err = vd.Validate(description, validator.StampPaletteDescriptionRule...); err != nil {
			return repository.ArgError("description", "Description must be 0-1000")
		}
		// スタンプ上限チェック
		// SqlのValuerが実装されていると、その結果でバリデーションをかけるため[]uuid.UUIDに変換
		uuids := stamps.ToUUIDSlice()
		if err = vd.Validate(uuids, validator.StampPaletteStampsRuleNotNil...); err != nil {
			return repository.ArgError("stamps", "stamps must be 0-200")
		}
		// スタンプ存在チェック
		if err = repo.ExistStamps(stamps); err != nil {
			return err
		}

		return tx.Create(stampPalette).Error
	})
	if err != nil {
		return nil, err
	}

	repo.hub.Publish(hub.Message{
		Name: event.StampPaletteCreated,
		Fields: hub.Fields{
			"user_id":          userID,
			"stamp_palette_id": stampPalette.ID,
			"stamp_palette":    stampPalette,
		},
	})
	return stampPalette, nil
}

// UpdateStampPalette implements StampPaletteRepository interface.
func (repo *Repository) UpdateStampPalette(id uuid.UUID, args repository.UpdateStampPaletteArgs) error {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	var userID uuid.UUID
	changes := map[string]interface{}{}
	err := repo.db.Transaction(func(tx *gorm.DB) error {
		var sp model.StampPalette
		if err := tx.First(&sp, &model.StampPalette{ID: id}).Error; err != nil {
			return convertError(err)
		}

		if args.Name.Valid {
			if err := vd.Validate(args.Name.V, validator.StampPaletteNameRuleRequired...); err != nil {
				return repository.ArgError("args.Name", "Name must be 1-30")
			}
			changes["name"] = args.Name.V
		}
		if args.Description.Valid {
			if err := vd.Validate(args.Description.V, validator.StampPaletteDescriptionRule...); err != nil {
				return repository.ArgError("args.Description", "Description must be 0-1000")
			}
			changes["description"] = args.Description.V
		}
		if args.Stamps != nil {
			uuids := args.Stamps.ToUUIDSlice()
			if err := vd.Validate(uuids, validator.StampPaletteStampsRuleNotNil...); err != nil {
				return repository.ArgError("args.Stamps", "stamps must be 0-200")
			}
			if err := repo.ExistStamps(args.Stamps); err != nil {
				return err
			}
			changes["stamps"] = args.Stamps
		}

		if len(changes) > 0 {
			return tx.Model(&sp).Updates(changes).Error
		}
		userID = sp.CreatorID
		return nil
	})
	if err != nil {
		return err
	}
	if len(changes) > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.StampPaletteUpdated,
			Fields: hub.Fields{
				"user_id":          userID,
				"stamp_palette_id": id,
			},
		})
	}
	return nil
}

// GetStampPalette implements StampPaletteRepository interface.
func (repo *Repository) GetStampPalette(id uuid.UUID) (sp *model.StampPalette, err error) {
	if id == uuid.Nil {
		return nil, repository.ErrNotFound
	}
	sp = &model.StampPalette{}
	if err := repo.db.Take(sp, &model.StampPalette{ID: id}).Error; err != nil {
		return nil, convertError(err)
	}
	return sp, nil
}

// DeleteStampPalette implements StampPaletteRepository interface.
func (repo *Repository) DeleteStampPalette(id uuid.UUID) (err error) {
	if id == uuid.Nil {
		return repository.ErrNilID
	}
	stampPalette, err := repo.GetStampPalette(id)
	if err != nil {
		return err
	}
	result := repo.db.Delete(&model.StampPalette{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.StampPaletteDeleted,
			Fields: hub.Fields{
				"user_id":          stampPalette.CreatorID,
				"stamp_palette_id": id,
			},
		})
		return nil
	}
	return repository.ErrNotFound
}

// GetStampPalettes implements StampPaletteRepository interface.
func (repo *Repository) GetStampPalettes(userID uuid.UUID) (sps []*model.StampPalette, err error) {
	sps = make([]*model.StampPalette, 0)
	tx := repo.db
	return sps, tx.Where("creator_id = ?", userID).Find(&sps).Error
}
