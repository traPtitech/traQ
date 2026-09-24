package notification

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/ws"
	"github.com/traPtitech/traQ/utils/optional"
)

const lastOnlinePersistTimeout = 5 * time.Second

func (ns *Service) startPresenceNotifications() {
	sub := ns.hub.Subscribe(200, event.UserOnline, event.UserOffline)
	go func() {
		for msg := range sub.Receiver {
			switch msg.Topic() {
			case event.UserOnline:
				userOnlineHandler(ns, msg)
			case event.UserOffline:
				userOfflineHandler(ns, msg)
			}
		}
	}()
}

func userOnlineHandler(ns *Service, ev hub.Message) {
	ns.ws.WriteMessage(
		"USER_ONLINE",
		map[string]interface{}{
			"id": ev.Fields["user_id"].(uuid.UUID),
		},
		ws.TargetAll(),
	)
}

func userOfflineHandler(ns *Service, ev hub.Message) {
	userID := ev.Fields["user_id"].(uuid.UUID)
	lastOnline := ev.Fields["datetime"].(time.Time).UTC().Truncate(time.Microsecond)

	ctx, cancel := context.WithTimeout(context.Background(), lastOnlinePersistTimeout)
	defer cancel()
	if err := ns.repo.UpdateUser(ctx, userID, repository.UpdateUserArgs{
		LastOnline: optional.From(lastOnline),
	}); err != nil {
		ns.logger.Error("failed to persist last online time", zap.Error(err), zap.Stringer("userId", userID))
	}

	ns.ws.WriteMessage(
		"USER_OFFLINE",
		map[string]interface{}{
			"id":         userID,
			"lastOnline": lastOnline,
		},
		ws.TargetAll(),
	)
}
