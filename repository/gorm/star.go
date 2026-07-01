package gorm

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/gormutil"
)

// AddStar implements StarRepository interface.
func (repo *Repository) AddStar(ctx context.Context, userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	var s model.Star
	result := repo.db.WithContext(ctx).FirstOrCreate(&s, &model.Star{UserID: userID, ChannelID: channelID})
	if result.Error != nil {
		if !gormutil.IsMySQLDuplicatedRecordErr(result.Error) {
			return result.Error
		}
	}
	repo.hub.Publish(hub.Message{
		Name: event.ChannelStared,
		Fields: hub.Fields{
			"user_id":    userID,
			"channel_id": channelID,
		},
	})
	return nil
}

// RemoveStar implements StarRepository interface.
func (repo *Repository) RemoveStar(ctx context.Context, userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	result := repo.db.WithContext(ctx).Delete(&model.Star{}, &model.Star{UserID: userID, ChannelID: channelID})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.ChannelUnstared,
			Fields: hub.Fields{
				"user_id":    userID,
				"channel_id": channelID,
			},
		})
	}
	return nil
}

// GetStaredChannels implements StarRepository interface.
func (repo *Repository) GetStaredChannels(ctx context.Context, userID uuid.UUID) (ids []uuid.UUID, err error) {
	ids = make([]uuid.UUID, 0)
	if userID == uuid.Nil {
		return ids, nil
	}
	return ids, repo.db.WithContext(ctx).Model(&model.Star{}).Where(&model.Star{UserID: userID}).Pluck("channel_id", &ids).Error
}
