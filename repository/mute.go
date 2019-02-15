package repository

import "github.com/satori/go.uuid"

type MuteRepository interface {
	MuteChannel(userID, channelID uuid.UUID) error
	UnmuteChannel(userID, channelID uuid.UUID) error
	GetMutedChannelIDs(userID uuid.UUID) ([]uuid.UUID, error)
	GetMuteUserIDs(channelID uuid.UUID) ([]uuid.UUID, error)
	IsChannelMuted(userID, channelID uuid.UUID) (bool, error)
}
