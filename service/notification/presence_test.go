package notification

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/ws"
)

type presenceRepository struct {
	repository.Repository
	update func(uuid.UUID, repository.UpdateUserArgs) error
}

func (r presenceRepository) UpdateUser(_ context.Context, id uuid.UUID, args repository.UpdateUserArgs) error {
	return r.update(id, args)
}

type presenceMessage struct {
	typeName string
	body     interface{}
}

type presenceWriter struct {
	messages []presenceMessage
}

func (w *presenceWriter) WriteMessage(typeName string, body interface{}, _ ws.TargetFunc) {
	w.messages = append(w.messages, presenceMessage{typeName, body})
}

func TestPresenceNotificationsPersistBeforeDelivery(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := hub.New()
		defer h.Close()
		userID := uuid.Must(uuid.NewV4())
		disconnectedAt := time.Date(2026, 9, 24, 10, 15, 0, 123456789, time.UTC)
		persistedAt := disconnectedAt.Truncate(time.Microsecond)
		resumeUpdate := make(chan struct{})
		resume := sync.OnceFunc(func() { close(resumeUpdate) })
		defer resume()
		var updatedUser uuid.UUID
		var updateArgs repository.UpdateUserArgs
		repo := presenceRepository{update: func(id uuid.UUID, args repository.UpdateUserArgs) error {
			updatedUser, updateArgs = id, args
			<-resumeUpdate
			return nil
		}}
		writer := &presenceWriter{}
		service := NewService(repo, nil, nil, nil, h, zap.NewNop(), nil, nil, nil, "")
		service.ws = writer

		h.Publish(hub.Message{Name: event.UserOffline, Fields: hub.Fields{
			"user_id": userID, "datetime": disconnectedAt,
		}})
		h.Publish(hub.Message{Name: event.UserOnline, Fields: hub.Fields{"user_id": userID}})
		synctest.Wait()
		require.Empty(t, writer.messages, "neither notification may overtake persistence")
		require.Equal(t, userID, updatedUser)
		require.True(t, updateArgs.LastOnline.Valid)
		require.Equal(t, persistedAt, updateArgs.LastOnline.V)

		resume()
		synctest.Wait()
		require.Len(t, writer.messages, 2)
		require.Equal(t, "USER_OFFLINE", writer.messages[0].typeName)
		require.Equal(t, "USER_ONLINE", writer.messages[1].typeName)
		payload, err := json.Marshal(writer.messages[0].body)
		require.NoError(t, err)
		require.JSONEq(t, `{"id":"`+userID.String()+`","lastOnline":"2026-09-24T10:15:00.123456Z"}`, string(payload))
	})
}

func TestPresenceNotificationsReportPersistenceFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := hub.New()
		defer h.Close()
		core, logs := observer.New(zap.ErrorLevel)
		repo := presenceRepository{update: func(uuid.UUID, repository.UpdateUserArgs) error {
			return errors.New("database unavailable")
		}}
		writer := &presenceWriter{}
		service := NewService(repo, nil, nil, nil, h, zap.New(core), nil, nil, nil, "")
		service.ws = writer
		userID := uuid.Must(uuid.NewV4())
		h.Publish(hub.Message{Name: event.UserOffline, Fields: hub.Fields{
			"user_id": userID, "datetime": time.Now(),
		}})
		h.Publish(hub.Message{Name: event.UserOnline, Fields: hub.Fields{"user_id": userID}})
		synctest.Wait()

		require.Equal(t, 1, logs.FilterMessage("failed to persist last online time").Len())
		require.Len(t, writer.messages, 2, "a database failure must not stop presence delivery")
		require.Equal(t, "USER_OFFLINE", writer.messages[0].typeName)
		require.Equal(t, "USER_ONLINE", writer.messages[1].typeName)
	})
}
