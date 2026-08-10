package gorm

import (
	"context"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/set"
)

// RegisterDevice implements DeviceRepository interface.
func (repo *Repository) RegisterDevice(ctx context.Context, userID uuid.UUID, token string) error {
	if userID == uuid.Nil {
		return repository.ErrNilID
	}
	if len(token) == 0 {
		return repository.ArgError("Token", "token is empty")
	}

	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var d model.Device
		if err := tx.First(&d, &model.Device{Token: token}).Error; err == nil {
			if d.UserID != userID {
				return repository.ArgError("Token", "the Token has already been associated with other user")
			}
			return nil
		} else if err != gorm.ErrRecordNotFound {
			return err
		}

		return tx.Create(&model.Device{
			Token:  token,
			UserID: userID,
		}).Error
	})
	return err
}

// GetDeviceTokens implements DeviceRepository interface.
func (repo *Repository) GetDeviceTokens(ctx context.Context, userIDs set.UUID) (tokens map[uuid.UUID][]string, err error) {
	var tmp []*model.Device
	if err := repo.db.WithContext(ctx).Where("user_id IN (?)", userIDs.StringArray()).Find(&tmp).Error; err != nil {
		return nil, err
	}

	tokens = make(map[uuid.UUID][]string, len(userIDs))
	for _, device := range tmp {
		tokens[device.UserID] = append(tokens[device.UserID], device.Token)
	}
	return tokens, nil
}

// DeleteDeviceTokens implements DeviceRepository interface.
func (repo *Repository) DeleteDeviceTokens(ctx context.Context, tokens []string) error {
	return repo.db.WithContext(ctx).Where("token IN (?)", tokens).Delete(&model.Device{}).Error
}
