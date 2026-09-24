package notification

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/repository/mock_repository"
	"github.com/traPtitech/traQ/service/ws"
	"github.com/traPtitech/traQ/testutils"
)

type presenceMessage struct {
	typeName string
	body     any
}

type presenceWriter struct {
	mu       sync.Mutex
	messages []presenceMessage
}

func (w *presenceWriter) WriteMessage(typeName string, body any, _ ws.TargetFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.messages = append(w.messages, presenceMessage{typeName, body})
}

func (w *presenceWriter) snapshot() []presenceMessage {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.messages)
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
		var updateArgs repository.UpdateUserArgs
		userRepo := mock_repository.NewMockUserRepository(gomock.NewController(t))
		userRepo.EXPECT().UpdateUser(gomock.Any(), userID, gomock.Any()).
			DoAndReturn(func(_ context.Context, _ uuid.UUID, args repository.UpdateUserArgs) error {
				updateArgs = args
				<-resumeUpdate
				return nil
			})
		repo := testutils.EmptyTestRepository{UserRepository: userRepo}
		writer := &presenceWriter{}
		service := NewService(repo, nil, nil, nil, h, zap.NewNop(), nil, nil, nil, "")
		service.messageWriter = writer

		h.Publish(hub.Message{Name: event.UserOffline, Fields: hub.Fields{
			"user_id": userID, "datetime": disconnectedAt,
		}})
		h.Publish(hub.Message{Name: event.UserOnline, Fields: hub.Fields{"user_id": userID}})
		synctest.Wait()
		assert.Empty(t, writer.snapshot(), "neither notification may overtake persistence")
		assert.True(t, updateArgs.LastOnline.Valid)
		assert.Equal(t, persistedAt, updateArgs.LastOnline.V)

		resume()
		synctest.Wait()
		messages := writer.snapshot()
		require.Len(t, messages, 2)
		assert.Equal(t, "USER_OFFLINE", messages[0].typeName)
		assert.Equal(t, "USER_ONLINE", messages[1].typeName)
		payload, err := json.Marshal(messages[0].body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"id":"`+userID.String()+`","lastOnline":"2026-09-24T10:15:00.123456Z"}`, string(payload))
	})
}

func TestPresenceNotificationsReportPersistenceFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := hub.New()
		defer h.Close()
		core, logs := observer.New(zap.ErrorLevel)
		userID := uuid.Must(uuid.NewV4())
		userRepo := mock_repository.NewMockUserRepository(gomock.NewController(t))
		userRepo.EXPECT().UpdateUser(gomock.Any(), userID, gomock.Any()).
			Return(errors.New("database unavailable"))
		repo := testutils.EmptyTestRepository{UserRepository: userRepo}
		writer := &presenceWriter{}
		service := NewService(repo, nil, nil, nil, h, zap.New(core), nil, nil, nil, "")
		service.messageWriter = writer
		h.Publish(hub.Message{Name: event.UserOffline, Fields: hub.Fields{
			"user_id": userID, "datetime": time.Now(),
		}})
		h.Publish(hub.Message{Name: event.UserOnline, Fields: hub.Fields{"user_id": userID}})
		synctest.Wait()

		assert.Equal(t, 1, logs.FilterMessage("failed to persist last online time").Len())
		messages := writer.snapshot()
		require.Len(t, messages, 2, "a database failure must not stop presence delivery")
		assert.Equal(t, "USER_OFFLINE", messages[0].typeName)
		assert.Equal(t, "USER_ONLINE", messages[1].typeName)
	})
}

func TestPresenceNotificationsContinueAfterPersistenceTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := hub.New()
		defer h.Close()
		stopUpdate := make(chan struct{})
		defer close(stopUpdate)
		core, logs := observer.New(zap.ErrorLevel)
		userID := uuid.Must(uuid.NewV4())
		var updateErr error
		userRepo := mock_repository.NewMockUserRepository(gomock.NewController(t))
		userRepo.EXPECT().UpdateUser(gomock.Any(), userID, gomock.Any()).
			DoAndReturn(func(ctx context.Context, _ uuid.UUID, _ repository.UpdateUserArgs) error {
				select {
				case <-ctx.Done():
					updateErr = ctx.Err()
					return updateErr
				case <-stopUpdate:
					return errors.New("test completed")
				}
			})
		repo := testutils.EmptyTestRepository{UserRepository: userRepo}
		writer := &presenceWriter{}
		service := NewService(repo, nil, nil, nil, h, zap.New(core), nil, nil, nil, "")
		service.messageWriter = writer
		h.Publish(hub.Message{Name: event.UserOffline, Fields: hub.Fields{
			"user_id": userID, "datetime": time.Now(),
		}})
		h.Publish(hub.Message{Name: event.UserOnline, Fields: hub.Fields{"user_id": userID}})
		synctest.Wait()

		time.Sleep(lastOnlinePersistTimeout - time.Nanosecond)
		synctest.Wait()
		assert.Empty(t, writer.snapshot(), "neither notification may overtake persistence")

		time.Sleep(time.Nanosecond)
		synctest.Wait()
		assert.ErrorIs(t, updateErr, context.DeadlineExceeded)
		entries := logs.FilterMessage("failed to persist last online time").All()
		require.Len(t, entries, 1)
		assert.Equal(t, context.DeadlineExceeded.Error(), entries[0].ContextMap()["error"])
		messages := writer.snapshot()
		require.Len(t, messages, 2, "a persistence timeout must not stop presence delivery")
		assert.Equal(t, "USER_OFFLINE", messages[0].typeName)
		assert.Equal(t, "USER_ONLINE", messages[1].typeName)
	})
}
