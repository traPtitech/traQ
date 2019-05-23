package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
)

// AddFavoriteStamp implements FavoriteStampRepository interface.
func (repo *GormRepository) AddFavoriteStamp(userID, stampID uuid.UUID) error {
	if userID == uuid.Nil || stampID == uuid.Nil {
		return ErrNilID
	}
	err := repo.transact(func(tx *gorm.DB) error {
		var s model.FavoriteStamp

		if exists, err := dbExists(tx, &model.Stamp{ID: stampID}); err != nil {
			return err
		} else if !exists {
			return ArgError("stampID", "the stamp doesn't exist")
		}

		return tx.FirstOrCreate(&s, &model.FavoriteStamp{UserID: userID, StampID: stampID}).Error
	})
	if err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.FavoriteStampAdded,
		Fields: hub.Fields{
			"user_id":  userID,
			"stamp_id": stampID,
		},
	})
	return nil
}

// RemoveFavoriteStamp implements FavoriteStampRepository interface.
func (repo *GormRepository) RemoveFavoriteStamp(userID, stampID uuid.UUID) error {
	if userID == uuid.Nil || stampID == uuid.Nil {
		return ErrNilID
	}
	result := repo.db.Delete(&model.FavoriteStamp{UserID: userID, StampID: stampID})
	if result.Error != nil {
		return result.Error
	}
	repo.hub.Publish(hub.Message{
		Name: event.FavoriteStampRemoved,
		Fields: hub.Fields{
			"user_id":  userID,
			"stamp_id": stampID,
		},
	})
	return nil
}

// GetUserFavoriteStamps implements FavoriteStampRepository interface.
func (repo *GormRepository) GetUserFavoriteStamps(userID uuid.UUID) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0)
	if userID == uuid.Nil {
		return ids, nil
	}
	return ids, repo.db.
		Model(&model.FavoriteStamp{}).
		Where(&model.FavoriteStamp{UserID: userID}).
		Order("created_at").
		Pluck("stamp_id", &ids).
		Error
}
