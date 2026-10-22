package channel

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
)

type uniqueChannelRepository struct {
	repository.ChannelRepository
	channels map[uuid.UUID]*model.Channel
}

func (r *uniqueChannelRepository) UpdateChannel(_ context.Context, id uuid.UUID, args repository.UpdateChannelArgs) (*model.Channel, error) {
	c := *r.channels[id]
	if args.Name.Valid {
		c.Name = args.Name.V
	}
	if args.Parent.Valid {
		c.ParentID = args.Parent.V
	}
	for otherID, other := range r.channels {
		if otherID != id && other.ParentID == c.ParentID && strings.EqualFold(other.Name, c.Name) {
			return nil, errors.New("duplicate channel name and parent")
		}
	}
	r.channels[id] = &c
	return &c, nil
}

func (*uniqueChannelRepository) RecordChannelEvent(context.Context, uuid.UUID, model.ChannelEventType, model.ChannelEventDetail, time.Time) error {
	return nil
}

func TestManagerImpl_ConcurrentChannelRenames(t *testing.T) {
	for range 10000 {
		a := &model.Channel{ID: uuid.Must(uuid.NewV7()), Name: "foo", IsPublic: true, IsVisible: true}
		b := &model.Channel{ID: uuid.Must(uuid.NewV7()), Name: "other", IsPublic: true, IsVisible: true}
		repo := &uniqueChannelRepository{channels: map[uuid.UUID]*model.Channel{a.ID: a, b.ID: b}}
		tree, err := makeChannelTree([]*model.Channel{a, b})
		require.NoError(t, err)
		m := &managerImpl{R: repo, T: tree, L: zap.NewNop(), MaxChannelDepth: 5}
		start := make(chan struct{})
		results := make(chan error, 3)
		var wg sync.WaitGroup
		for _, op := range []struct {
			id   uuid.UUID
			name string
		}{{a.ID, "bar"}, {b.ID, "foo"}, {a.ID, "foo"}} {
			wg.Go(func() {
				<-start
				results <- m.UpdateChannel(context.Background(), op.id, repository.UpdateChannelArgs{Name: optional.From(op.name)})
			})
		}
		close(start)
		wg.Wait()
		m.P.Wait()
		close(results)
		for err := range results {
			if err != nil {
				require.ErrorIs(t, err, ErrChannelNameConflicts)
			}
		}
	}
}
