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
func (repo *Repository) RegisterDevice(ctx context.Context, userID uuid.UUID, args repository.RegisterDeviceArgs) error {
	if userID == uuid.Nil {
		return repository.ErrNilID
	}

	token := args.FID.V
	tokenType := model.DeviceTokenTypeFID
	if !args.FID.Valid {
		token = args.Token.V
		tokenType = model.DeviceTokenTypeToken
		if !args.Token.Valid {
			return repository.ArgError("Token, FID", "token and fid are empty")
		}
	}

	err := repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var d model.Device
		if len(args.Token.ValueOrZero()) != 0 && len(args.FID.ValueOrZero()) != 0 {
			if err := tx.Delete(&model.Device{Token: args.Token.ValueOrZero(), TokenType: model.DeviceTokenTypeToken}).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
			}
		}

		if err := tx.First(&d, &model.Device{Token: token, TokenType: tokenType}).Error; err == nil {
			if d.UserID != userID {
				return repository.ArgError("Token", "the Token has already been associated with other user")
			}
			return nil
		} else if err != gorm.ErrRecordNotFound {
			return err
		}

		return tx.Create(&model.Device{
			Token:     token,
			TokenType: tokenType,
			UserID:    userID,
		}).Error
	})
	return err
}

// GetDeviceTokens implements DeviceRepository interface.
func (repo *Repository) GetDeviceTokens(ctx context.Context, userIDs set.UUID) (tokens map[uuid.UUID][]repository.TokenEntry, err error) {
	var tmp []*model.Device
	if err := repo.db.WithContext(ctx).Where("user_id IN (?)", userIDs.StringArray()).Find(&tmp).Error; err != nil {
		return nil, err
	}

	tokens = make(map[uuid.UUID][]repository.TokenEntry, len(userIDs))
	for _, device := range tmp {
		tokens[device.UserID] = append(tokens[device.UserID], repository.TokenEntry{Token: device.Token, TokenType: device.TokenType})
	}
	return tokens, nil
}

// DeleteDeviceTokens implements DeviceRepository interface.
func (repo *Repository) DeleteDeviceTokens(ctx context.Context, tokens []repository.RegisterDeviceArgs) error {
	tokenValues := make([]string, len(tokens))
	for i, t := range tokens {
		if t.Token.Valid {
			tokenValues[i] = t.Token.ValueOrZero()
		} else {
			tokenValues[i] = t.FID.ValueOrZero()
		}
	}
	return repo.db.WithContext(ctx).Where("token IN (?)", tokenValues).Delete(&model.Device{}).Error
}
