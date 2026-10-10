package scheduled

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/testutils"
)

type channelsForTest struct {
	channel.Manager
	ch             *model.Channel
	accessible     bool
	parentArchived bool
	members        []uuid.UUID
	err            error
}

func (c channelsForTest) GetChannel(context.Context, uuid.UUID) (*model.Channel, error) {
	return c.ch, c.err
}
func (c channelsForTest) IsChannelAccessibleToUser(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return c.accessible, nil
}
func (c channelsForTest) GetDMChannelMembers(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return c.members, nil
}
func (c channelsForTest) PublicChannelTree(context.Context) channel.Tree {
	return treeForTest{archived: c.parentArchived}
}

type treeForTest struct {
	channel.Tree
	archived bool
}

func (t treeForTest) IsArchivedChannel(uuid.UUID) bool { return t.archived }

func TestValidateRechecksCurrentPermissionsAndDMRecipients(t *testing.T) {
	ctx := context.Background()
	for _, name := range []string{"public", "dm", "inactive", "permission revoked", "inaccessible", "parent archived", "channel missing", "transient failure"} {
		t.Run(name, func(t *testing.T) {
			repo := testutils.NewTestRepository()
			u, err := repo.CreateUser(ctx, repository.CreateUserArgs{Name: "scheduler", Role: role.User})
			require.NoError(t, err)
			cid, other := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
			cm := channelsForTest{ch: &model.Channel{ID: cid, IsPublic: true, IsVisible: true}, accessible: true, members: []uuid.UUID{u.GetID(), other}}
			want := ""
			switch name {
			case "dm":
				cm.ch.IsPublic = false
			case "inactive":
				user := repo.Users[u.GetID()]
				user.Status = model.UserAccountStatusDeactivated
				repo.Users[u.GetID()] = user
				want = "user_inactive"
			case "permission revoked":
				user := repo.Users[u.GetID()]
				user.Role = "unknown-role"
				repo.Users[u.GetID()] = user
				want = "permission_revoked"
			case "inaccessible":
				cm.accessible = false
				want = "channel_unavailable"
			case "parent archived":
				cm.parentArchived = true
				want = "channel_archived"
			case "channel missing":
				cm.err = channel.ErrChannelNotFound
				want = "channel_unavailable"
			case "transient failure":
				cm.err = errors.New("database offline")
			}
			w := Worker{Repo: repo, Channels: cm, RBAC: testutils.NewTestRBAC()}
			recipients, failure, err := w.Validate(ctx, &model.ScheduledMessage{UserID: u.GetID(), ChannelID: cid})
			if name == "transient failure" {
				require.Error(t, err)
				require.Empty(t, failure)
				return
			}
			require.NoError(t, err)
			require.Equal(t, want, failure)
			if name == "dm" {
				require.ElementsMatch(t, []uuid.UUID{u.GetID(), other}, recipients)
			}
			if name == "public" {
				require.Contains(t, recipients, uuid.Nil)
			}
		})
	}
}
