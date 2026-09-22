//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package channel

import (
	"context"
	"errors"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

var (
	ErrChannelNotFound      = errors.New("channel not found")
	ErrChannelNameConflicts = errors.New("channel name conflicts")
	ErrInvalidChannelName   = errors.New("invalid channel name")
	ErrInvalidChannelPath   = errors.New("invalid channel path")
	ErrInvalidParentChannel = errors.New("invalid parent channel")
	ErrTooDeepChannel       = errors.New("too deep channel")
	ErrChannelArchived      = errors.New("channel archived")
	ErrForcedNotification   = errors.New("forced notification channel")
	ErrInvalidChannel       = errors.New("invalid channel")
)

type Manager interface {
	GetChannel(ctx context.Context, id uuid.UUID) (*model.Channel, error)
	GetChannelPathFromID(ctx context.Context, id uuid.UUID) string
	GetChannelFromPath(ctx context.Context, path string) (*model.Channel, error)
	CreatePublicChannel(ctx context.Context, name string, parent, creatorID uuid.UUID) (*model.Channel, error)
	UpdateChannel(ctx context.Context, id uuid.UUID, args repository.UpdateChannelArgs) error
	PublicChannelTree(ctx context.Context) Tree

	ChangeChannelSubscriptions(ctx context.Context, channelID uuid.UUID, subscriptions map[uuid.UUID]model.ChannelSubscribeLevel, keepOffLevel bool, updaterID uuid.UUID) error

	ArchiveChannel(ctx context.Context, id uuid.UUID, updaterID uuid.UUID) error
	UnarchiveChannel(ctx context.Context, id uuid.UUID, updaterID uuid.UUID) error

	GetDMChannel(ctx context.Context, user1, user2 uuid.UUID) (*model.Channel, error)
	GetDMChannelMembers(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error)
	GetDMChannelMapping(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]uuid.UUID, error)

	IsChannelAccessibleToUser(ctx context.Context, userID, channelID uuid.UUID) (bool, error)
	IsPublicChannel(ctx context.Context, id uuid.UUID) bool

	// Wait マネージャーの処理の完了を待ちます
	Wait()
}
