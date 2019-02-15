package impl

import (
	"github.com/leandro-lugaresi/hub"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

// AddStar チャンネルをお気に入り登録します
func (repo *RepositoryImpl) AddStar(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	var s model.Star
	result := repo.db.FirstOrCreate(&s, &model.Star{UserID: userID, ChannelID: channelID})
	if result.Error != nil {
		return result.Error
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

// RemoveStar チャンネルのお気に入りを解除します
func (repo *RepositoryImpl) RemoveStar(userID, channelID uuid.UUID) error {
	if userID == uuid.Nil || channelID == uuid.Nil {
		return repository.ErrNilID
	}
	result := repo.db.Where(&model.Star{UserID: userID, ChannelID: channelID}).Delete(&model.Star{})
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

// GetStaredChannels ユーザーがお気に入りをしているチャンネルIDを取得する
func (repo *RepositoryImpl) GetStaredChannels(userID uuid.UUID) (ids []uuid.UUID, err error) {
	ids = make([]uuid.UUID, 0)
	if userID == uuid.Nil {
		return ids, nil
	}
	err = repo.db.
		Model(&model.Star{}).
		Where(&model.Star{UserID: userID}).
		Pluck("channel_id", &ids).
		Error
	return ids, err
}
