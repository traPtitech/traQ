package notification

import (
	"context"
	"encoding/json"
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

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/repository/mock_repository"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/channel/mock_channel"
	"github.com/traPtitech/traQ/service/fcm"
	msvc "github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/qall"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/ws"
	"github.com/traPtitech/traQ/testutils"
	"github.com/traPtitech/traQ/utils/message"
	"github.com/traPtitech/traQ/utils/set"
)

const testOrigin = "https://example.com"

// ---- 入力イベントの組み立て ----

func messageCreatedEvent(m *model.Message, parsed *message.ParseResult) hub.Message {
	return hub.Message{Name: event.MessageCreated, Fields: hub.Fields{
		"message_id":   m.ID,
		"message":      m,
		"parse_result": parsed,
	}}
}

func messageUpdatedEvent(oldM, newM *model.Message) hub.Message {
	return hub.Message{Name: event.MessageUpdated, Fields: hub.Fields{
		"message_id":  newM.ID,
		"old_message": oldM,
		"message":     newM,
	}}
}

func messageDeletedEvent(m *model.Message, unreads []*model.Unread) hub.Message {
	return hub.Message{Name: event.MessageDeleted, Fields: hub.Fields{
		"message_id":      m.ID,
		"message":         m,
		"deleted_unreads": unreads,
	}}
}

func messageStampedEvent(messageID, stampID, userID uuid.UUID, count int, createdAt time.Time) hub.Message {
	return hub.Message{Name: event.MessageStamped, Fields: hub.Fields{
		"message_id": messageID,
		"stamp_id":   stampID,
		"user_id":    userID,
		"count":      count,
		"created_at": createdAt,
	}}
}

func messageUnstampedEvent(messageID, stampID, userID uuid.UUID) hub.Message {
	return hub.Message{Name: event.MessageUnstamped, Fields: hub.Fields{
		"message_id": messageID,
		"stamp_id":   stampID,
		"user_id":    userID,
	}}
}

func messagePinEvent(topic string, messageID, channelID uuid.UUID) hub.Message {
	return hub.Message{Name: topic, Fields: hub.Fields{
		"message_id": messageID,
		"channel_id": channelID,
	}}
}

func channelPrivateEvent(topic string, ch *model.Channel) hub.Message {
	return hub.Message{Name: topic, Fields: hub.Fields{
		"channel_id": ch.ID,
		"private":    !ch.IsPublic,
	}}
}

func channelUserEvent(topic string, userID, channelID uuid.UUID) hub.Message {
	f := hub.Fields{
		"user_id":    userID,
		"channel_id": channelID,
	}
	if topic == event.ChannelRead {
		f["read_messages_num"] = 1
	}
	return hub.Message{Name: topic, Fields: f}
}

func channelSubscribersChangedEvent(channelID uuid.UUID, subscriberIDs []uuid.UUID) hub.Message {
	return hub.Message{Name: event.ChannelSubscribersChanged, Fields: hub.Fields{
		"channel_id":     channelID,
		"subscriber_ids": subscriberIDs,
	}}
}

func stampPaletteEvent(topic string, userID, paletteID uuid.UUID) hub.Message {
	f := hub.Fields{
		"user_id":          userID,
		"stamp_palette_id": paletteID,
	}
	if topic == event.StampPaletteCreated {
		f["stamp_palette"] = &model.StampPalette{ID: paletteID, CreatorID: userID}
	}
	return hub.Message{Name: topic, Fields: f}
}

func clipFolderEvent(topic string, userID, folderID uuid.UUID) hub.Message {
	folder := &model.ClipFolder{ID: folderID, OwnerID: userID}
	f := hub.Fields{
		"user_id":        userID,
		"clip_folder_id": folderID,
		"clip_folder":    folder,
	}
	if topic == event.ClipFolderUpdated {
		f["old_clip_folder"] = folder
	}
	return hub.Message{Name: topic, Fields: f}
}

func clipFolderMessageEvent(topic string, userID, folderID, messageID uuid.UUID) hub.Message {
	return hub.Message{Name: topic, Fields: hub.Fields{
		"user_id":                userID,
		"clip_folder_id":         folderID,
		"clip_folder_message_id": messageID,
		"clip_folder_message":    &model.ClipFolderMessage{FolderID: folderID, MessageID: messageID},
	}}
}

func channelCreatedEvent(ch *model.Channel) hub.Message {
	return hub.Message{Name: event.ChannelCreated, Fields: hub.Fields{
		"channel_id": ch.ID,
		"channel":    ch,
		"private":    !ch.IsPublic,
	}}
}

func userTagEvent(topic string, userID, tagID uuid.UUID) hub.Message {
	return hub.Message{Name: topic, Fields: hub.Fields{
		"user_id": userID,
		"tag_id":  tagID,
	}}
}

func userGroupUpdatedEvent(groupID uuid.UUID) hub.Message {
	return hub.Message{Name: event.UserGroupUpdated, Fields: hub.Fields{
		"group_id": groupID,
	}}
}

func userGroupMemberAddedEvent(groupID, userID uuid.UUID) hub.Message {
	return userGroupMemberEvent(event.UserGroupMemberAdded, groupID, userID)
}

// userGroupMemberEvent メンバー・管理者の追加・変更・削除のイベント (Fields は group_id と user_id)
func userGroupMemberEvent(topic string, groupID, userID uuid.UUID) hub.Message {
	return hub.Message{Name: topic, Fields: hub.Fields{
		"group_id": groupID,
		"user_id":  userID,
	}}
}

func userCreatedEvent(u *model.User) hub.Message {
	return hub.Message{Name: event.UserCreated, Fields: hub.Fields{
		"user_id": u.ID,
		"user":    u,
	}}
}

func stampCreatedEvent(stamp *model.Stamp) hub.Message {
	return hub.Message{Name: event.StampCreated, Fields: event.FieldsOf(event.StampCreatedPayload{
		StampID: stamp.ID,
		Stamp:   stamp,
	})}
}

func stampUpdatedEvent(stampID uuid.UUID) hub.Message {
	return hub.Message{Name: event.StampUpdated, Fields: event.FieldsOf(event.StampUpdatedPayload{
		StampID: stampID,
	})}
}

func stampDeletedEvent(stampID uuid.UUID) hub.Message {
	return hub.Message{Name: event.StampDeleted, Fields: event.FieldsOf(event.StampDeletedPayload{
		StampID: stampID,
	})}
}

func qallRoomStateChangedEvent(states []qall.RoomWithParticipants) hub.Message {
	return hub.Message{Name: event.QallRoomStateChanged, Fields: hub.Fields{
		"roomStates": states,
	}}
}

// ---- テスト用の fake ----

type recordedMessage struct {
	typeName string
	body     string // JSON
	target   ws.TargetFunc
}

type recordingWriter struct {
	mu       sync.Mutex
	messages []recordedMessage
}

func (w *recordingWriter) WriteMessage(typeName string, body any, target ws.TargetFunc) {
	b, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.messages = append(w.messages, recordedMessage{typeName: typeName, body: string(b), target: target})
}

func (w *recordingWriter) snapshot() []recordedMessage {
	w.mu.Lock()
	defer w.mu.Unlock()
	return slices.Clone(w.messages)
}

type fcmSend struct {
	targets set.UUID
	payload fcm.Payload
}

type recordingFCM struct {
	mu    sync.Mutex
	sends []fcmSend
}

func (c *recordingFCM) Send(targets set.UUID, p *fcm.Payload, _ bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sends = append(c.sends, fcmSend{targets: targets.Clone(), payload: *p})
}

func (c *recordingFCM) Close() {}

func (c *recordingFCM) snapshot() []fcmSend {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.sends)
}

type fakeSession struct {
	key       string
	userID    uuid.UUID
	viewing   uuid.UUID
	timeline  bool
	viewState viewer.State
}

func (s fakeSession) Key() string                          { return s.key }
func (s fakeSession) UserID() uuid.UUID                    { return s.userID }
func (s fakeSession) ViewState() (uuid.UUID, viewer.State) { return s.viewing, s.viewState }
func (s fakeSession) TimelineStreaming() bool              { return s.timeline }

// targetKeys target に該当するセッションのキーを返す
func targetKeys(target ws.TargetFunc, sessions ...fakeSession) []string {
	keys := make([]string, 0)
	for _, s := range sessions {
		if target(s) {
			keys = append(keys, s.key)
		}
	}
	return keys
}

type fakeMessageManager struct {
	msvc.Manager
	channels map[uuid.UUID]uuid.UUID // messageID -> channelID
}

func (m *fakeMessageManager) Get(_ context.Context, id uuid.UUID) (msvc.Message, error) {
	cid, ok := m.channels[id]
	if !ok {
		return nil, msvc.ErrNotFound
	}
	return fakeMessage{id: id, channelID: cid}, nil
}

type fakeMessage struct {
	msvc.Message
	id        uuid.UUID
	channelID uuid.UUID
}

func (m fakeMessage) GetID() uuid.UUID        { return m.id }
func (m fakeMessage) GetChannelID() uuid.UUID { return m.channelID }

type notificationDeps struct {
	repo testutils.EmptyTestRepository
	cm   *mock_channel.MockManager
	mm   msvc.Manager
	fcm  *recordingFCM
	vm   *viewer.Manager
}

func newID() uuid.UUID { return uuid.Must(uuid.NewV4()) }

// startService synctest のバブル内で notification を起動し、hub と writer を返す。
// 呼び出し側は defer h.Close() で購読 goroutine を終了させること。
func startService(deps notificationDeps) (*hub.Hub, *recordingWriter) {
	h := hub.New()
	writer := &recordingWriter{}
	var fcmClient fcm.Client
	if deps.fcm != nil {
		fcmClient = deps.fcm
	} else {
		fcmClient = fcm.NewNullClient()
	}
	var cm channel.Manager
	if deps.cm != nil {
		cm = deps.cm
	}
	newService(deps.repo, cm, deps.mm, nil, h, zap.NewNop(), fcmClient, writer, deps.vm, testOrigin)
	return h, writer
}

func messagesOfType(messages []recordedMessage, typeName string) []recordedMessage {
	var res []recordedMessage
	for _, m := range messages {
		if m.typeName == typeName {
			res = append(res, m)
		}
	}
	return res
}

// messagesByBody body ごとに 1 件ずつであることを確認し、body -> メッセージ の map を返す
func messagesByBody(t *testing.T, messages []recordedMessage) map[string]recordedMessage {
	t.Helper()
	res := make(map[string]recordedMessage, len(messages))
	for _, m := range messages {
		require.NotContainsf(t, res, m.body, "duplicated body: %s", m.body)
		res[m.body] = m
	}
	return res
}

// ---- MESSAGE_CREATED ----

// TestMessageCreatedNotification メッセージ作成時の Notification が正しく動いているかをテストする
func TestMessageCreatedNotification(t *testing.T) {
	// viewer.Manager は GC 用の goroutine を常駐させるため、synctest のバブルの外で作る
	vm := viewer.NewManager(hub.New())

	t.Run("public channel", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			chID := newID()
			author := &model.User{ID: newID(), Name: "author", DisplayName: "Author", Status: model.UserAccountStatusActive}
			notifyUserID, markUserID, otherUserID := newID(), newID(), newID()
			m := &model.Message{ID: newID(), UserID: author.ID, ChannelID: chID, Text: "hello"}

			tree := mock_channel.NewMockTree(ctrl)
			tree.EXPECT().IsChannelPresent(chID).Return(true).AnyTimes()
			tree.EXPECT().IsForceChannel(chID).Return(false).AnyTimes()
			tree.EXPECT().GetChannelPath(chID).Return("general").AnyTimes()
			cm := mock_channel.NewMockManager(ctrl)
			cm.EXPECT().PublicChannelTree(gomock.Any()).Return(tree).AnyTimes()

			q := repository.UsersQuery{}.Active().NotBot()
			userRepo := mock_repository.NewMockUserRepository(ctrl)
			userRepo.EXPECT().GetUser(gomock.Any(), author.ID, false).Return(author, nil)
			userRepo.EXPECT().GetUserIDs(gomock.Any(), q.SubscriberAtNotifyLevelOf(chID)).Return([]uuid.UUID{notifyUserID}, nil)
			userRepo.EXPECT().GetUserIDs(gomock.Any(), q.SubscriberAtMarkLevelOf(chID)).Return([]uuid.UUID{markUserID, author.ID}, nil)
			msgRepo := mock_repository.NewMockMessageRepository(ctrl)
			msgRepo.EXPECT().SetMessageUnreads(gomock.Any(), map[uuid.UUID]bool{markUserID: false}, m.ID).Return(nil)

			fcmClient := &recordingFCM{}
			h, writer := startService(notificationDeps{
				repo: testutils.EmptyTestRepository{UserRepository: userRepo, MessageRepository: msgRepo},
				cm:   cm,
				fcm:  fcmClient,
				vm:   vm,
			})
			defer h.Close()
			h.Publish(messageCreatedEvent(m, message.Parse(m.Text)))
			synctest.Wait()

			sessions := []fakeSession{
				{key: "author", userID: author.ID},
				{key: "mark", userID: markUserID},
				{key: "notify", userID: notifyUserID},
				{key: "timeline", userID: otherUserID, timeline: true},
				{key: "other", userID: otherUserID},
			}
			messages := messagesOfType(writer.snapshot(), "MESSAGE_CREATED")
			require.Len(t, messages, 2)
			byBody := messagesByBody(t, messages)
			notCiting := `{"id":"` + m.ID.String() + `","is_citing":false}`
			citing := `{"id":"` + m.ID.String() + `","is_citing":true}`
			require.Contains(t, byBody, notCiting)
			require.Contains(t, byBody, citing)
			assert.ElementsMatch(t, []string{"mark", "timeline"}, targetKeys(byBody[notCiting].target, sessions...))
			assert.Empty(t, targetKeys(byBody[citing].target, sessions...))

			sends := fcmClient.snapshot()
			require.Len(t, sends, 1)
			assert.ElementsMatch(t, []uuid.UUID{notifyUserID}, sends[0].targets.Array())
			assert.Equal(t, "new_message", sends[0].payload.Type)
			assert.Equal(t, "#general", sends[0].payload.Title)
			assert.Equal(t, "/channels/general", sends[0].payload.Path)
			assert.Equal(t, "Author: hello", sends[0].payload.Body)
			assert.Equal(t, testOrigin+"/api/v3/public/icon/author", sends[0].payload.Icon)
			assert.Equal(t, "c:"+chID.String(), sends[0].payload.Tag)
		})
	})

	t.Run("direct message", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			chID := newID()
			author := &model.User{ID: newID(), Name: "author", DisplayName: "Author", Status: model.UserAccountStatusActive}
			peerID, otherUserID := newID(), newID()
			m := &model.Message{ID: newID(), UserID: author.ID, ChannelID: chID, Text: "hi"}

			tree := mock_channel.NewMockTree(ctrl)
			tree.EXPECT().IsChannelPresent(chID).Return(false).AnyTimes()
			tree.EXPECT().IsForceChannel(chID).Return(false).AnyTimes()
			cm := mock_channel.NewMockManager(ctrl)
			cm.EXPECT().PublicChannelTree(gomock.Any()).Return(tree).AnyTimes()

			q := repository.UsersQuery{}.Active().NotBot()
			userRepo := mock_repository.NewMockUserRepository(ctrl)
			userRepo.EXPECT().GetUser(gomock.Any(), author.ID, false).Return(author, nil)
			userRepo.EXPECT().GetUserIDs(gomock.Any(), q.CMemberOf(chID)).Return([]uuid.UUID{author.ID, peerID}, nil)
			msgRepo := mock_repository.NewMockMessageRepository(ctrl)
			msgRepo.EXPECT().SetMessageUnreads(gomock.Any(), map[uuid.UUID]bool{peerID: false}, m.ID).Return(nil)

			fcmClient := &recordingFCM{}
			h, writer := startService(notificationDeps{
				repo: testutils.EmptyTestRepository{UserRepository: userRepo, MessageRepository: msgRepo},
				cm:   cm,
				fcm:  fcmClient,
				vm:   vm,
			})
			defer h.Close()
			h.Publish(messageCreatedEvent(m, message.Parse(m.Text)))
			synctest.Wait()

			sessions := []fakeSession{
				{key: "author", userID: author.ID},
				{key: "peer", userID: peerID},
				{key: "timeline", userID: otherUserID, timeline: true},
			}
			messages := messagesOfType(writer.snapshot(), "MESSAGE_CREATED")
			require.Len(t, messages, 2)
			byBody := messagesByBody(t, messages)
			notCiting := `{"id":"` + m.ID.String() + `","is_citing":false}`
			citing := `{"id":"` + m.ID.String() + `","is_citing":true}`
			require.Contains(t, byBody, notCiting)
			require.Contains(t, byBody, citing)
			assert.ElementsMatch(t, []string{"author", "peer"}, targetKeys(byBody[notCiting].target, sessions...))
			assert.Empty(t, targetKeys(byBody[citing].target, sessions...))

			sends := fcmClient.snapshot()
			require.Len(t, sends, 1)
			assert.ElementsMatch(t, []uuid.UUID{peerID}, sends[0].targets.Array())
			assert.Equal(t, "@Author", sends[0].payload.Title)
			assert.Equal(t, "/users/author", sends[0].payload.Path)
			assert.Equal(t, "hi", sends[0].payload.Body)
		})
	})
}

// ---- MESSAGE_UPDATED / MESSAGE_DELETED ----

// TestMessageUpdatedAndDeletedNotification メッセージ更新・削除時の Notification が正しく動いているかをテストする
func TestMessageUpdatedAndDeletedNotification(t *testing.T) {
	cases := []struct {
		name      string
		typeName  string
		public    bool
		makeEvent func(m *model.Message) hub.Message
		want      []string
	}{
		{
			name:     "updated in public channel",
			typeName: "MESSAGE_UPDATED",
			public:   true,
			makeEvent: func(m *model.Message) hub.Message {
				return messageUpdatedEvent(&model.Message{ID: m.ID, ChannelID: m.ChannelID}, m)
			},
			want: []string{"viewer", "timeline"},
		},
		{
			name:     "updated in DM",
			typeName: "MESSAGE_UPDATED",
			public:   false,
			makeEvent: func(m *model.Message) hub.Message {
				return messageUpdatedEvent(&model.Message{ID: m.ID, ChannelID: m.ChannelID}, m)
			},
			want: []string{"viewer"},
		},
		{
			name:      "deleted in public channel",
			typeName:  "MESSAGE_DELETED",
			public:    true,
			makeEvent: func(m *model.Message) hub.Message { return messageDeletedEvent(m, nil) },
			want:      []string{"viewer", "timeline"},
		},
		{
			name:      "deleted in DM",
			typeName:  "MESSAGE_DELETED",
			public:    false,
			makeEvent: func(m *model.Message) hub.Message { return messageDeletedEvent(m, nil) },
			want:      []string{"viewer"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				chID := newID()
				m := &model.Message{ID: newID(), UserID: newID(), ChannelID: chID, Text: "edited"}
				cm := mock_channel.NewMockManager(gomock.NewController(t))
				cm.EXPECT().IsPublicChannel(gomock.Any(), chID).Return(tc.public)

				h, writer := startService(notificationDeps{cm: cm})
				defer h.Close()
				h.Publish(tc.makeEvent(m))
				synctest.Wait()

				messages := writer.snapshot()
				require.Len(t, messages, 1)
				assert.Equal(t, tc.typeName, messages[0].typeName)
				assert.JSONEq(t, `{"id":"`+m.ID.String()+`"}`, messages[0].body)
				sessions := []fakeSession{
					{key: "viewer", userID: newID(), viewing: chID},
					{key: "timeline", userID: newID(), timeline: true},
					{key: "other", userID: newID(), viewing: newID()},
				}
				assert.ElementsMatch(t, tc.want, targetKeys(messages[0].target, sessions...))
			})
		})
	}
}

// ---- MESSAGE_STAMPED ----

// TestMessageStampedNotification メッセージにスタンプが押されたときの Notification が正しく動いているかをテストする
func TestMessageStampedNotification(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		chID, messageID, stampID, userID := newID(), newID(), newID(), newID()
		createdAt := time.Date(2026, 11, 4, 12, 34, 56, 0, time.UTC)
		mm := &fakeMessageManager{channels: map[uuid.UUID]uuid.UUID{messageID: chID}}

		h, writer := startService(notificationDeps{mm: mm})
		defer h.Close()
		h.Publish(messageStampedEvent(messageID, stampID, userID, 3, createdAt))
		synctest.Wait()

		messages := writer.snapshot()
		require.Len(t, messages, 1)
		assert.Equal(t, "MESSAGE_STAMPED", messages[0].typeName)
		assert.JSONEq(t, `{
			"message_id":"`+messageID.String()+`",
			"user_id":"`+userID.String()+`",
			"stamp_id":"`+stampID.String()+`",
			"count":3,
			"created_at":"2026-11-04T12:34:56Z"
		}`, messages[0].body)
		sessions := []fakeSession{
			{key: "viewer", userID: newID(), viewing: chID},
			{key: "timeline", userID: newID(), timeline: true},
		}
		assert.ElementsMatch(t, []string{"viewer"}, targetKeys(messages[0].target, sessions...))
	})
}

// TestMessageUnstampedNotification メッセージからスタンプが消されたときの Notification が正しく動いているかをテストする
func TestMessageUnstampedNotification(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		chID, messageID, stampID, userID := newID(), newID(), newID(), newID()
		mm := &fakeMessageManager{channels: map[uuid.UUID]uuid.UUID{messageID: chID}}

		h, writer := startService(notificationDeps{mm: mm})
		defer h.Close()
		h.Publish(messageUnstampedEvent(messageID, stampID, userID))
		synctest.Wait()

		messages := writer.snapshot()
		require.Len(t, messages, 1)
		assert.Equal(t, "MESSAGE_UNSTAMPED", messages[0].typeName)
		assert.JSONEq(t, `{
			"message_id":"`+messageID.String()+`",
			"user_id":"`+userID.String()+`",
			"stamp_id":"`+stampID.String()+`"
		}`, messages[0].body)
		sessions := []fakeSession{
			{key: "viewer", userID: newID(), viewing: chID},
			{key: "timeline", userID: newID(), timeline: true},
		}
		assert.ElementsMatch(t, []string{"viewer"}, targetKeys(messages[0].target, sessions...))
	})
}

// ---- MESSAGE_PINNED / MESSAGE_UNPINNED ----

// TestMessagePinNotification メッセージのピン留め・解除時の Notification が正しく動いているかをテストする
func TestMessagePinNotification(t *testing.T) {
	cases := map[string]string{
		event.MessagePinned:   "MESSAGE_PINNED",
		event.MessageUnpinned: "MESSAGE_UNPINNED",
	}
	for topic, wantType := range cases {
		t.Run(topic, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				chID, messageID := newID(), newID()

				h, writer := startService(notificationDeps{})
				defer h.Close()
				h.Publish(messagePinEvent(topic, messageID, chID))
				synctest.Wait()

				messages := writer.snapshot()
				require.Len(t, messages, 1)
				assert.Equal(t, wantType, messages[0].typeName)
				assert.JSONEq(t, `{
					"message_id":"`+messageID.String()+`",
					"channel_id":"`+chID.String()+`"
				}`, messages[0].body)
				sessions := []fakeSession{
					{key: "viewer", userID: newID(), viewing: chID},
					{key: "other", userID: newID(), viewing: newID()},
					{key: "timeline", userID: newID(), timeline: true},
				}
				assert.ElementsMatch(t, []string{"viewer"}, targetKeys(messages[0].target, sessions...))
			})
		})
	}
}

// ---- CHANNEL_CREATED ----

// TestChannelCreatedNotification チャンネル作成時の Notification が正しく動いているかをテストする
func TestChannelCreatedNotification(t *testing.T) {
	t.Run("public channel", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ch := &model.Channel{ID: newID(), Name: "general", IsPublic: true}

			h, writer := startService(notificationDeps{})
			defer h.Close()
			h.Publish(channelCreatedEvent(ch))
			synctest.Wait()

			messages := writer.snapshot()
			require.Len(t, messages, 1)
			assert.Equal(t, "CHANNEL_CREATED", messages[0].typeName)
			assert.JSONEq(t, `{"id":"`+ch.ID.String()+`"}`, messages[0].body)
			assert.ElementsMatch(t, []string{"a", "b"}, targetKeys(messages[0].target,
				fakeSession{key: "a", userID: newID()}, fakeSession{key: "b", userID: newID()}))
		})
	})

	t.Run("DM channel", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ch := &model.Channel{ID: newID(), IsPublic: false}
			u1, u2 := newID(), newID()
			cm := mock_channel.NewMockManager(gomock.NewController(t))
			cm.EXPECT().GetDMChannelMembers(gomock.Any(), ch.ID).Return([]uuid.UUID{u1, u2}, nil)

			h, writer := startService(notificationDeps{cm: cm})
			defer h.Close()
			h.Publish(channelCreatedEvent(ch))
			synctest.Wait()

			messages := writer.snapshot()
			require.Len(t, messages, 2)
			sessions := []fakeSession{
				{key: "u1", userID: u1},
				{key: "u2", userID: u2},
				{key: "other", userID: newID()},
			}
			got := map[string][]string{}
			for _, msg := range messages {
				assert.Equal(t, "CHANNEL_CREATED", msg.typeName)
				got[msg.body] = targetKeys(msg.target, sessions...)
			}
			assert.Equal(t, map[string][]string{
				`{"dm_user_id":"` + u1.String() + `","id":"` + ch.ID.String() + `"}`: {"u2"},
				`{"dm_user_id":"` + u2.String() + `","id":"` + ch.ID.String() + `"}`: {"u1"},
			}, got)
		})
	})
}

// ---- CHANNEL_UPDATED / CHANNEL_DELETED ----

// TestChannelUpdatedAndDeletedNotification チャンネル更新・削除時の Notification が正しく動いているかをテストする
func TestChannelUpdatedAndDeletedNotification(t *testing.T) {
	cases := map[string]string{
		event.ChannelUpdated: "CHANNEL_UPDATED",
		event.ChannelDeleted: "CHANNEL_DELETED",
	}
	for topic, wantType := range cases {
		t.Run(topic+" public channel", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ch := &model.Channel{ID: newID(), IsPublic: true}

				h, writer := startService(notificationDeps{})
				defer h.Close()
				h.Publish(channelPrivateEvent(topic, ch))
				synctest.Wait()

				messages := writer.snapshot()
				require.Len(t, messages, 1)
				assert.Equal(t, wantType, messages[0].typeName)
				assert.JSONEq(t, `{"id":"`+ch.ID.String()+`"}`, messages[0].body)
				assert.ElementsMatch(t, []string{"a", "b"}, targetKeys(messages[0].target,
					fakeSession{key: "a", userID: newID()}, fakeSession{key: "b", userID: newID()}))
			})
		})
		t.Run(topic+" DM channel", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ch := &model.Channel{ID: newID(), IsPublic: false}
				u := newID()
				cm := mock_channel.NewMockManager(gomock.NewController(t))
				cm.EXPECT().GetDMChannelMembers(gomock.Any(), ch.ID).Return([]uuid.UUID{u}, nil)

				h, writer := startService(notificationDeps{cm: cm})
				defer h.Close()
				h.Publish(channelPrivateEvent(topic, ch))
				synctest.Wait()

				messages := writer.snapshot()
				require.Len(t, messages, 1)
				assert.Equal(t, wantType, messages[0].typeName)
				assert.JSONEq(t, `{"id":"`+ch.ID.String()+`","dm_user_id":"`+u.String()+`"}`, messages[0].body)
				assert.ElementsMatch(t, []string{"u"}, targetKeys(messages[0].target,
					fakeSession{key: "u", userID: u}, fakeSession{key: "other", userID: newID()}))
			})
		})
	}
}

// ---- CHANNEL_STARED / CHANNEL_UNSTARED / MESSAGE_READ ----

// TestChannelUserNotification チャンネルのスター・既読時の Notification が正しく動いているかをテストする
func TestChannelUserNotification(t *testing.T) {
	cases := map[string]string{
		event.ChannelStared:   "CHANNEL_STARED",
		event.ChannelUnstared: "CHANNEL_UNSTARED",
		event.ChannelRead:     "MESSAGE_READ",
	}
	for topic, wantType := range cases {
		t.Run(topic, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				userID, chID := newID(), newID()

				h, writer := startService(notificationDeps{})
				defer h.Close()
				h.Publish(channelUserEvent(topic, userID, chID))
				synctest.Wait()

				messages := writer.snapshot()
				require.Len(t, messages, 1)
				assert.Equal(t, wantType, messages[0].typeName)
				assert.JSONEq(t, `{"id":"`+chID.String()+`"}`, messages[0].body)
				assert.ElementsMatch(t, []string{"user"}, targetKeys(messages[0].target,
					fakeSession{key: "user", userID: userID}, fakeSession{key: "other", userID: newID(), viewing: chID}))
			})
		})
	}
}

// ---- CHANNEL_SUBSCRIBERS_CHANGED ----

// TestChannelSubscribersChangedNotification チャンネルの購読者変更時の Notification が正しく動いているかをテストする
func TestChannelSubscribersChangedNotification(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		chID, subscriber := newID(), newID()

		h, writer := startService(notificationDeps{})
		defer h.Close()
		h.Publish(channelSubscribersChangedEvent(chID, []uuid.UUID{subscriber}))
		synctest.Wait()

		messages := writer.snapshot()
		require.Len(t, messages, 1)
		assert.Equal(t, "CHANNEL_SUBSCRIBERS_CHANGED", messages[0].typeName)
		assert.JSONEq(t, `{"id":"`+chID.String()+`"}`, messages[0].body)
		assert.ElementsMatch(t, []string{"viewer", "subscriber"}, targetKeys(messages[0].target,
			fakeSession{key: "viewer", userID: newID(), viewing: chID},
			fakeSession{key: "subscriber", userID: subscriber},
			fakeSession{key: "other", userID: newID()}))
	})
}

// ---- STAMP_PALETTE_* ----

// TestStampPaletteNotification スタンプパレットの作成・更新・削除時の Notification が正しく動いているかをテストする
func TestStampPaletteNotification(t *testing.T) {
	cases := map[string]string{
		event.StampPaletteCreated: "STAMP_PALETTE_CREATED",
		event.StampPaletteUpdated: "STAMP_PALETTE_UPDATED",
		event.StampPaletteDeleted: "STAMP_PALETTE_DELETED",
	}
	for topic, wantType := range cases {
		t.Run(topic, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				userID, paletteID := newID(), newID()

				h, writer := startService(notificationDeps{})
				defer h.Close()
				h.Publish(stampPaletteEvent(topic, userID, paletteID))
				synctest.Wait()

				messages := writer.snapshot()
				require.Len(t, messages, 1)
				assert.Equal(t, wantType, messages[0].typeName)
				assert.JSONEq(t, `{"id":"`+paletteID.String()+`"}`, messages[0].body)
				assert.ElementsMatch(t, []string{"user"}, targetKeys(messages[0].target,
					fakeSession{key: "user", userID: userID}, fakeSession{key: "other", userID: newID()}))
			})
		})
	}
}

// ---- CLIP_FOLDER_* ----

// TestClipFolderNotification クリップフォルダーの作成・更新・削除時の Notification が正しく動いているかをテストする
func TestClipFolderNotification(t *testing.T) {
	cases := map[string]string{
		event.ClipFolderCreated: "CLIP_FOLDER_CREATED",
		event.ClipFolderUpdated: "CLIP_FOLDER_UPDATED",
		event.ClipFolderDeleted: "CLIP_FOLDER_DELETED",
	}
	for topic, wantType := range cases {
		t.Run(topic, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				userID, folderID := newID(), newID()

				h, writer := startService(notificationDeps{})
				defer h.Close()
				h.Publish(clipFolderEvent(topic, userID, folderID))
				synctest.Wait()

				messages := writer.snapshot()
				require.Len(t, messages, 1)
				assert.Equal(t, wantType, messages[0].typeName)
				assert.JSONEq(t, `{"id":"`+folderID.String()+`"}`, messages[0].body)
				assert.ElementsMatch(t, []string{"user"}, targetKeys(messages[0].target,
					fakeSession{key: "user", userID: userID}, fakeSession{key: "other", userID: newID()}))
			})
		})
	}
}

// TestClipFolderMessageNotification クリップフォルダーへのメッセージの追加・除外時の Notification が正しく動いているかをテストする
func TestClipFolderMessageNotification(t *testing.T) {
	cases := map[string]string{
		event.ClipFolderMessageAdded:   "CLIP_FOLDER_MESSAGE_ADDED",
		event.ClipFolderMessageDeleted: "CLIP_FOLDER_MESSAGE_DELETED",
	}
	for topic, wantType := range cases {
		t.Run(topic, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				userID, folderID, messageID := newID(), newID(), newID()

				h, writer := startService(notificationDeps{})
				defer h.Close()
				h.Publish(clipFolderMessageEvent(topic, userID, folderID, messageID))
				synctest.Wait()

				messages := writer.snapshot()
				require.Len(t, messages, 1)
				assert.Equal(t, wantType, messages[0].typeName)
				assert.JSONEq(t, `{"folder_id":"`+folderID.String()+`","message_id":"`+messageID.String()+`"}`, messages[0].body)
				assert.ElementsMatch(t, []string{"user"}, targetKeys(messages[0].target,
					fakeSession{key: "user", userID: userID}, fakeSession{key: "other", userID: newID()}))
			})
		})
	}
}

// ---- 複数トピックを 1 つの WS イベントにまとめるもの ----

// TestUserTagsUpdatedNotification ユーザーのタグ変更時の Notification が正しく動いているかをテストする
func TestUserTagsUpdatedNotification(t *testing.T) {
	for _, topic := range []string{event.UserTagAdded, event.UserTagUpdated, event.UserTagRemoved} {
		t.Run(topic, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				userID, tagID := newID(), newID()

				h, writer := startService(notificationDeps{})
				defer h.Close()
				h.Publish(userTagEvent(topic, userID, tagID))
				synctest.Wait()

				messages := writer.snapshot()
				require.Len(t, messages, 1)
				assert.Equal(t, "USER_TAGS_UPDATED", messages[0].typeName)
				assert.JSONEq(t, `{"id":"`+userID.String()+`","tag_id":"`+tagID.String()+`"}`, messages[0].body)
				assert.ElementsMatch(t, []string{"any"}, targetKeys(messages[0].target, fakeSession{key: "any", userID: newID()}))
			})
		})
	}
}

// TestUserGroupUpdatedNotification ユーザーグループ変更時の Notification が正しく動いているかをテストする
func TestUserGroupUpdatedNotification(t *testing.T) {
	cases := map[string]func(groupID uuid.UUID) hub.Message{
		event.UserGroupUpdated:     userGroupUpdatedEvent,
		event.UserGroupMemberAdded: func(groupID uuid.UUID) hub.Message { return userGroupMemberAddedEvent(groupID, newID()) },
	}
	for _, topic := range []string{event.UserGroupMemberUpdated, event.UserGroupMemberRemoved, event.UserGroupAdminAdded, event.UserGroupAdminRemoved} {
		cases[topic] = func(groupID uuid.UUID) hub.Message { return userGroupMemberEvent(topic, groupID, newID()) }
	}
	for topic, makeEvent := range cases {
		t.Run(topic, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				groupID := newID()

				h, writer := startService(notificationDeps{})
				defer h.Close()
				h.Publish(makeEvent(groupID))
				synctest.Wait()

				messages := writer.snapshot()
				require.Len(t, messages, 1)
				assert.Equal(t, "USER_GROUP_UPDATED", messages[0].typeName)
				assert.JSONEq(t, `{"id":"`+groupID.String()+`"}`, messages[0].body)
				assert.ElementsMatch(t, []string{"any"}, targetKeys(messages[0].target, fakeSession{key: "any", userID: newID()}))
			})
		})
	}
}

// ---- USER_JOINED ----

// TestUserJoinedNotification ユーザー作成時の Notification が正しく動いているかをテストする
func TestUserJoinedNotification(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		u := &model.User{ID: newID(), Name: "newcomer"}

		h, writer := startService(notificationDeps{})
		defer h.Close()
		h.Publish(userCreatedEvent(u))
		synctest.Wait()

		messages := writer.snapshot()
		require.Len(t, messages, 1)
		assert.Equal(t, "USER_JOINED", messages[0].typeName)
		assert.JSONEq(t, `{"id":"`+u.ID.String()+`"}`, messages[0].body)
		assert.ElementsMatch(t, []string{"any"}, targetKeys(messages[0].target, fakeSession{key: "any", userID: newID()}))
	})
}

// ---- STAMP_CREATED / STAMP_UPDATED / STAMP_DELETED ----

// TestStampNotification スタンプ作成・更新・削除時の Notification が正しく動いているかをテストする
func TestStampNotification(t *testing.T) {
	cases := []struct {
		typeName  string
		makeEvent func(stampID uuid.UUID) hub.Message
	}{
		{"STAMP_CREATED", func(stampID uuid.UUID) hub.Message {
			return stampCreatedEvent(&model.Stamp{ID: stampID, Name: "test", FileID: newID()})
		}},
		{"STAMP_UPDATED", stampUpdatedEvent},
		{"STAMP_DELETED", stampDeletedEvent},
	}
	for _, tc := range cases {
		t.Run(tc.typeName, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				stampID := newID()

				h, writer := startService(notificationDeps{})
				defer h.Close()
				h.Publish(tc.makeEvent(stampID))
				synctest.Wait()

				messages := writer.snapshot()
				require.Len(t, messages, 1)
				assert.Equal(t, tc.typeName, messages[0].typeName)
				assert.JSONEq(t, `{"id":"`+stampID.String()+`"}`, messages[0].body)
				assert.ElementsMatch(t, []string{"any"}, targetKeys(messages[0].target, fakeSession{key: "any", userID: newID()}))
			})
		})
	}
}

// ---- QALL_ROOM_STATE_CHANGED ----

// TestQallRoomStateChangedNotification Qall のルーム状態変更時の Notification が正しく動いているかをテストする
func TestQallRoomStateChangedNotification(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		roomID := newID()
		joinedAt := time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC)
		states := []qall.RoomWithParticipants{{
			RoomID:    roomID,
			IsWebinar: true,
			Participants: []qall.Participant{{
				Identity:   "user_1",
				Name:       "User",
				JoinedAt:   joinedAt,
				CanPublish: true,
			}},
		}}

		h, writer := startService(notificationDeps{})
		defer h.Close()
		h.Publish(qallRoomStateChangedEvent(states))
		synctest.Wait()

		messages := writer.snapshot()
		require.Len(t, messages, 1)
		assert.Equal(t, "QALL_ROOM_STATE_CHANGED", messages[0].typeName)
		// swagger (room_id / state) とは異なるが、既存実装の roomStates (全ルームの配列) を正とする
		assert.JSONEq(t, `{"roomStates":[{
			"roomId":"`+roomID.String()+`",
			"isWebinar":true,
			"participants":[{"identity":"user_1","name":"User","joinedAt":"2026-11-04T09:00:00Z","canPublish":true}]
		}]}`, messages[0].body)
		assert.ElementsMatch(t, []string{"any"}, targetKeys(messages[0].target, fakeSession{key: "any", userID: newID()}))
	})
}
