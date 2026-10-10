package gorm

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
)

func makeReservation(t *testing.T, repo repository.Repository, user model.UserInfo, ch *model.Channel) *model.ScheduledMessage {
	t.Helper()
	m := &model.ScheduledMessage{
		ID: uuid.Must(uuid.NewV7()), UserID: user.GetID(), ChannelID: ch.ID,
		Content: "scheduled notice", DraftContent: "scheduled notice", ScheduledAt: time.Now().Add(-time.Hour),
	}
	require.NoError(t, repo.CreateScheduledMessage(context.Background(), m))
	t.Cleanup(func() {
		require.NoError(t, repo.(*Repository).db.Delete(&model.ScheduledMessage{}, "id = ?", m.ID).Error)
	})
	return m
}

func makePrivateUpload(t *testing.T, repo repository.Repository, user model.UserInfo) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	require.NoError(t, repo.SaveFileMeta(context.Background(), &model.FileMeta{
		ID: id, Name: "notice.png", Mime: "image/png", Size: 100, CreatorID: optional.From(user.GetID()), Type: model.FileTypeUserFile,
	}, []*model.FileACLEntry{{UserID: user.GetID(), Allow: true}}))
	return id
}

func TestScheduledMessageDeliveryIsAtomicAndPrivate(t *testing.T) {
	r, _, _, user, ch := setupWithUserAndChannel(t, common, false)
	repo := r.(*Repository)
	other := mustMakeUser(t, r, rand, false)
	ctx := context.Background()
	fid := makePrivateUpload(t, r, user)
	m := &model.ScheduledMessage{
		ID: uuid.Must(uuid.NewV7()), UserID: user.GetID(), ChannelID: ch.ID,
		Content: "notice with image", DraftContent: "notice", ScheduledAt: time.Now().Add(-time.Hour),
		Attachments: []model.ScheduledMessageFile{{FileID: fid}},
	}
	require.NoError(t, r.CreateScheduledMessage(ctx, m))
	visible, err := r.IsFileAccessible(ctx, fid, other.GetID())
	require.NoError(t, err)
	require.False(t, visible)
	list, err := r.GetScheduledMessages(ctx, other.GetID())
	require.NoError(t, err)
	require.Empty(t, list)
	_, err = r.CancelScheduledMessage(ctx, m.ID, other.GetID())
	require.ErrorIs(t, err, repository.ErrNotFound)

	sub := repo.hub.Subscribe(10, event.MessageCreated)
	defer repo.hub.Unsubscribe(sub)
	var validated atomic.Int32
	validate := func(_ context.Context, _ *model.ScheduledMessage) ([]uuid.UUID, string, error) {
		validated.Add(1)
		return []uuid.UUID{uuid.Nil, user.GetID()}, "", nil
	}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Go(func() { errs <- r.ProcessScheduledMessage(ctx, m.ID, validate) })
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, validated.Load())
	var count int64
	require.NoError(t, repo.db.Model(&model.Message{}).Where("channel_id = ? AND user_id = ?", ch.ID, user.GetID()).Count(&count).Error)
	require.EqualValues(t, 1, count)
	visible, err = r.IsFileAccessible(ctx, fid, other.GetID())
	require.NoError(t, err)
	require.True(t, visible)
	meta, err := r.GetFileMeta(ctx, fid)
	require.NoError(t, err)
	require.Equal(t, ch.ID, meta.ChannelID.V)
	_, err = r.CancelScheduledMessage(ctx, m.ID, user.GetID())
	require.ErrorIs(t, err, repository.ErrAlreadyExists)
	select {
	case <-sub.Receiver:
	default:
		t.Fatal("delivery did not publish the normal message event")
	}
	select {
	case <-sub.Receiver:
		t.Fatal("duplicate message event")
	default:
	}
}

func TestScheduledMessageCancellationWinsAgainstWorker(t *testing.T) {
	r, _, _, user, ch := setupWithUserAndChannel(t, common, false)
	ctx := context.Background()
	m := makeReservation(t, r, user, ch)
	_, err := r.CancelScheduledMessage(ctx, m.ID, user.GetID())
	require.NoError(t, err)
	require.NoError(t, r.ProcessScheduledMessage(ctx, m.ID, func(context.Context, *model.ScheduledMessage) ([]uuid.UUID, string, error) {
		t.Fatal("a canceled reservation must not reach validation or delivery")
		return nil, "", nil
	}))
	list, err := r.GetScheduledMessages(ctx, user.GetID())
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestScheduledMessageRollbackKeepsAttachmentsPrivate(t *testing.T) {
	r, _, _, user, ch := setupWithUserAndChannel(t, common, false)
	repo := r.(*Repository)
	ctx := context.Background()
	fid := makePrivateUpload(t, r, user)
	m := &model.ScheduledMessage{
		ID: uuid.Must(uuid.NewV7()), UserID: user.GetID(), ChannelID: ch.ID,
		Content: uuid.Must(uuid.NewV7()).String(), ScheduledAt: time.Now().Add(-time.Minute),
		Attachments: []model.ScheduledMessageFile{{FileID: fid}},
	}
	require.NoError(t, r.CreateScheduledMessage(ctx, m))
	trigger := "reject_" + strings.ReplaceAll(m.ID.String(), "-", "")
	require.NoError(t, repo.db.Exec("CREATE TRIGGER "+trigger+" BEFORE INSERT ON messages FOR EACH ROW BEGIN IF NEW.text = '"+m.Content+"' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'injected failure'; END IF; END").Error)
	t.Cleanup(func() { require.NoError(t, repo.db.Exec("DROP TRIGGER "+trigger).Error) })
	err := r.ProcessScheduledMessage(ctx, m.ID, func(context.Context, *model.ScheduledMessage) ([]uuid.UUID, string, error) {
		return []uuid.UUID{uuid.Nil, user.GetID()}, "", nil
	})
	require.Error(t, err)
	visible, err := r.IsFileAccessible(ctx, fid, uuid.Must(uuid.NewV7()))
	require.NoError(t, err)
	require.False(t, visible)
	list, err := r.GetScheduledMessages(ctx, user.GetID())
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, model.ScheduledMessagePending, list[0].Status)
}

func TestScheduledMessageFailuresAndFutureReservations(t *testing.T) {
	r, _, _, user, ch := setupWithUserAndChannel(t, common, false)
	ctx := context.Background()
	m := makeReservation(t, r, user, ch)
	retryErr := errors.New("temporary database failure")
	err := r.ProcessScheduledMessage(ctx, m.ID, func(context.Context, *model.ScheduledMessage) ([]uuid.UUID, string, error) {
		return nil, "", retryErr
	})
	require.ErrorIs(t, err, retryErr)
	require.NoError(t, r.ProcessScheduledMessage(ctx, m.ID, func(context.Context, *model.ScheduledMessage) ([]uuid.UUID, string, error) {
		return nil, "channel_archived", nil
	}))
	list, err := r.GetScheduledMessages(ctx, user.GetID())
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "channel_archived", list[0].Failure)
	require.Equal(t, model.ScheduledMessageFailed, list[0].Status)
	// Failed items do not block subsequent reservations.
	next := makeReservation(t, r, user, ch)
	require.NoError(t, r.(*Repository).db.Model(next).Update("scheduled_at", time.Now().Add(time.Hour)).Error)
	require.NoError(t, r.ProcessScheduledMessage(ctx, next.ID, func(context.Context, *model.ScheduledMessage) ([]uuid.UUID, string, error) {
		t.Fatal("future reservation must not be dispatched")
		return nil, "", nil
	}))
}

func TestScheduledMessageCannotReuseOrPublishAnotherUsersUpload(t *testing.T) {
	r, _, _, user, ch := setupWithUserAndChannel(t, common, false)
	other := mustMakeUser(t, r, rand, false)
	ctx := context.Background()
	fid := makePrivateUpload(t, r, user)
	m := &model.ScheduledMessage{
		ID: uuid.Must(uuid.NewV7()), UserID: other.GetID(), ChannelID: ch.ID,
		Content: "notice", ScheduledAt: time.Now().Add(time.Hour), Attachments: []model.ScheduledMessageFile{{FileID: fid}},
	}
	require.ErrorIs(t, r.CreateScheduledMessage(ctx, m), repository.ErrForbidden)
	m.UserID = user.GetID()
	require.NoError(t, r.CreateScheduledMessage(ctx, m))
	m.ID = uuid.Must(uuid.NewV7())
	require.ErrorIs(t, r.CreateScheduledMessage(ctx, m), repository.ErrAlreadyExists)
}

func TestScheduledMessageMissingAttachmentFailsDelivery(t *testing.T) {
	r, _, _, user, ch := setupWithUserAndChannel(t, common, false)
	ctx := context.Background()
	fid := makePrivateUpload(t, r, user)
	m := &model.ScheduledMessage{
		ID: uuid.Must(uuid.NewV7()), UserID: user.GetID(), ChannelID: ch.ID,
		Content: "notice with attachment", ScheduledAt: time.Now().Add(-time.Minute),
		Attachments: []model.ScheduledMessageFile{{FileID: fid}},
	}
	require.NoError(t, r.CreateScheduledMessage(ctx, m))
	require.NoError(t, r.DeleteFileMeta(ctx, fid))
	require.NoError(t, r.ProcessScheduledMessage(ctx, m.ID, func(context.Context, *model.ScheduledMessage) ([]uuid.UUID, string, error) {
		return []uuid.UUID{uuid.Nil, user.GetID()}, "", nil
	}))
	list, err := r.GetScheduledMessages(ctx, user.GetID())
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "attachment_unavailable", list[0].Failure)
	require.Equal(t, []uuid.UUID{fid}, list[0].FileIDs())
	var count int64
	require.NoError(t, r.(*Repository).db.Model(&model.Message{}).Where("channel_id = ?", ch.ID).Count(&count).Error)
	require.Zero(t, count)
}

func TestScheduledMessageStoresAbsoluteTime(t *testing.T) {
	r, _, _, user, ch := setupWithUserAndChannel(t, common, false)
	ctx := context.Background()
	instant := time.Now().Add(time.Hour).Truncate(time.Second)
	m := &model.ScheduledMessage{
		ID: uuid.Must(uuid.NewV7()), UserID: user.GetID(), ChannelID: ch.ID,
		Content: "timezone notice", ScheduledAt: instant.In(time.FixedZone("JST", 9*60*60)),
	}
	require.NoError(t, r.CreateScheduledMessage(ctx, m))
	list, err := r.GetScheduledMessages(ctx, user.GetID())
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.True(t, list[0].ScheduledAt.Equal(instant))
	ids, err := r.GetDueScheduledMessageIDs(ctx, instant.Add(time.Second).In(time.FixedZone("PDT", -7*60*60)), 10000)
	require.NoError(t, err)
	require.Contains(t, ids, m.ID)
}

func TestScheduledMessageLimitAndCancellationRace(t *testing.T) {
	r, _, _, user, ch := setupWithUserAndChannel(t, common, false)
	ctx := context.Background()
	var first *model.ScheduledMessage
	for range repository.ScheduledMessageLimit {
		m := makeReservation(t, r, user, ch)
		if first == nil {
			first = m
		}
	}
	require.Error(t, r.CreateScheduledMessage(ctx, &model.ScheduledMessage{
		ID: uuid.Must(uuid.NewV7()), UserID: user.GetID(), ChannelID: ch.ID, Content: "over limit", ScheduledAt: time.Now(),
	}))
	cancelResult, processResult := make(chan error, 1), make(chan error, 1)
	go func() { _, err := r.CancelScheduledMessage(ctx, first.ID, user.GetID()); cancelResult <- err }()
	go func() {
		processResult <- r.ProcessScheduledMessage(ctx, first.ID, func(context.Context, *model.ScheduledMessage) ([]uuid.UUID, string, error) {
			return []uuid.UUID{uuid.Nil, user.GetID()}, "", nil
		})
	}()
	cancelErr := <-cancelResult
	require.NoError(t, <-processResult)
	var count int64
	require.NoError(t, r.(*Repository).db.Model(&model.Message{}).Where("channel_id = ?", ch.ID).Count(&count).Error)
	if cancelErr == nil {
		require.Zero(t, count)
	} else {
		require.ErrorIs(t, cancelErr, repository.ErrAlreadyExists)
		require.EqualValues(t, 1, count)
	}
	// Both sent and canceled reservations release capacity.
	makeReservation(t, r, user, ch)
}
