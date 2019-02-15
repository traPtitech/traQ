package repository

import (
	"errors"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
)

var (
	// ErrChannelDepthLimitation チャンネルの深さ制限を超えている
	ErrChannelDepthLimitation = errors.New("channel depth limit exceeded")
)

// ChannelRepository チャンネルリポジトリ
type ChannelRepository interface {
	CreatePublicChannel(name string, parent, creatorID uuid.UUID) (*model.Channel, error)
	CreatePrivateChannel(name string, creatorID uuid.UUID, members []uuid.UUID) (*model.Channel, error)
	CreateChildChannel(name string, parentID, creatorID uuid.UUID) (*model.Channel, error)
	UpdateChannelAttributes(channelID uuid.UUID, visibility, forced *bool) error
	UpdateChannelTopic(channelID uuid.UUID, topic string, updaterID uuid.UUID) error
	ChangeChannelName(channelID uuid.UUID, name string) error
	ChangeChannelParent(channelID, parent uuid.UUID) error
	DeleteChannel(channelID uuid.UUID) error
	GetChannel(channelID uuid.UUID) (*model.Channel, error)
	GetChannelByMessageID(messageID uuid.UUID) (*model.Channel, error)
	GetChannelsByUserID(userID uuid.UUID) ([]*model.Channel, error)
	GetDirectMessageChannel(user1, user2 uuid.UUID) (*model.Channel, error)
	GetAllChannels() ([]*model.Channel, error)
	IsChannelPresent(name string, parent uuid.UUID) (bool, error)
	IsChannelAccessibleToUser(userID, channelID uuid.UUID) (bool, error)
	GetParentChannel(channelID uuid.UUID) (*model.Channel, error)
	GetChildrenChannelIDs(channelID uuid.UUID) ([]uuid.UUID, error)
	GetDescendantChannelIDs(channelID uuid.UUID) ([]uuid.UUID, error)
	GetAscendantChannelIDs(channelID uuid.UUID) ([]uuid.UUID, error)
	GetChannelPath(id uuid.UUID) (string, error)
	GetChannelDepth(id uuid.UUID) (int, error)
	AddPrivateChannelMember(channelID, userID uuid.UUID) error
	GetPrivateChannelMemberIDs(channelID uuid.UUID) ([]uuid.UUID, error)
	IsUserPrivateChannelMember(channelID, userID uuid.UUID) (bool, error)
	SubscribeChannel(userID, channelID uuid.UUID) error
	UnsubscribeChannel(userID, channelID uuid.UUID) error
	GetSubscribingUserIDs(channelID uuid.UUID) ([]uuid.UUID, error)
	GetSubscribedChannelIDs(userID uuid.UUID) ([]uuid.UUID, error)
}
