package scheduled

import (
	"context"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/permission"
)

type Worker struct {
	Repo     repository.Repository
	Channels channel.Manager
	RBAC     rbac.RBAC
	Logger   *zap.Logger
}

// Run catches up immediately on startup and retries transient failures every five seconds.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if err := w.DeliverDue(ctx); err != nil && ctx.Err() == nil {
			w.Logger.Error("failed to deliver scheduled messages", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) DeliverDue(ctx context.Context) error {
	ids, err := w.Repo.GetDueScheduledMessageIDs(ctx, time.Now(), 100)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := w.Repo.ProcessScheduledMessage(ctx, id, w.Validate); err != nil && !errors.Is(err, repository.ErrNotFound) {
			w.Logger.Error("failed to deliver scheduled message", zap.Stringer("id", id), zap.Error(err))
		}
	}
	return nil
}

// Validate is also used when accepting a reservation; current permissions are
// checked again inside the delivery transaction, rather than saving credentials.
func (w *Worker) Validate(ctx context.Context, m *model.ScheduledMessage) ([]uuid.UUID, string, error) {
	u, err := w.Repo.GetUser(ctx, m.UserID, false)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, "user_unavailable", nil
	}
	if err != nil {
		return nil, "", err
	}
	if u.GetState() != model.UserAccountStatusActive {
		return nil, "user_inactive", nil
	}
	if !w.RBAC.IsGranted(u.GetRole(), permission.PostMessage) {
		return nil, "permission_revoked", nil
	}
	if len(m.Attachments) > 0 && !w.RBAC.IsGranted(u.GetRole(), permission.UploadFile) {
		return nil, "permission_revoked", nil
	}
	ch, err := w.Channels.GetChannel(ctx, m.ChannelID)
	if errors.Is(err, channel.ErrChannelNotFound) || errors.Is(err, repository.ErrNotFound) {
		return nil, "channel_unavailable", nil
	}
	if err != nil {
		return nil, "", err
	}
	if ch.IsArchived() || (ch.IsPublic && w.Channels.PublicChannelTree(ctx).IsArchivedChannel(ch.ID)) {
		return nil, "channel_archived", nil
	}
	accessible, err := w.Channels.IsChannelAccessibleToUser(ctx, m.UserID, m.ChannelID)
	if err != nil {
		return nil, "", err
	}
	if !accessible {
		return nil, "channel_unavailable", nil
	}
	if ch.IsPublic {
		return []uuid.UUID{uuid.Nil, m.UserID}, "", nil
	}
	members, err := w.Channels.GetDMChannelMembers(ctx, ch.ID)
	return members, "", err
}
