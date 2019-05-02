package repository

import (
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
)

// MuteChannel implements MuteRepository interface.
func (repo *GormRepository) MuteChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return ErrNilID
	}
	var m model.Mute
	if err := repo.db.FirstOrCreate(&m, &model.Mute{UserID: userID, ChannelID: channelID}).Error; err != nil {
		return err
	}
	repo.hub.Publish(hub.Message{
		Name: event.ChannelMuted,
		Fields: hub.Fields{
			"user_id":    userID,
			"channel_id": channelID,
		},
	})
	return nil
}

// UnmuteChannel implements MuteRepository interface.
func (repo *GormRepository) UnmuteChannel(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return ErrNilID
	}
	result := repo.db.Delete(&model.Mute{UserID: userID, ChannelID: channelID})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		repo.hub.Publish(hub.Message{
			Name: event.ChannelUnmuted,
			Fields: hub.Fields{
				"user_id":    userID,
				"channel_id": channelID,
			},
		})
	}
	return nil
}

// GetMutedChannelIDs implements MuteRepository interface.
func (repo *GormRepository) GetMutedChannelIDs(userID uuid.UUID) (ids []uuid.UUID, err error) {
	ids = make([]uuid.UUID, 0)
	if userID == uuid.Nil {
		return ids, nil
	}
	return ids, dbPluck(repo.db, &model.Mute{UserID: userID}, "channel_id", &ids)
}

// GetMuteUserIDs implements MuteRepository interface.
func (repo *GormRepository) GetMuteUserIDs(channelID uuid.UUID) (ids []uuid.UUID, err error) {
	ids = make([]uuid.UUID, 0)
	if channelID == uuid.Nil {
		return ids, nil
	}
	return ids, dbPluck(repo.db, &model.Mute{ChannelID: channelID}, "user_id", &ids)
}

// IsChannelMuted implements MuteRepository interface.
func (repo *GormRepository) IsChannelMuted(userID, channelID uuid.UUID) (bool, error) {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return false, nil
	}
	return dbExists(repo.db, &model.Mute{UserID: userID, ChannelID: channelID})
}
