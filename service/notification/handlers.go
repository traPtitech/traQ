package notification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/traPtitech/traQ/utils/optional"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/fcm"
	"github.com/traPtitech/traQ/service/qall"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/ws"
	"github.com/traPtitech/traQ/utils/message"
	"github.com/traPtitech/traQ/utils/set"
)

type eventHandler func(ns *Service, ev hub.Message)

var handlerMap = map[string]eventHandler{
	event.MessageCreated:            messageCreatedHandler,
	event.MessageUpdated:            messageUpdatedHandler,
	event.MessageDeleted:            messageDeletedHandler,
	event.MessagePinned:             messagePinnedHandler,
	event.MessageUnpinned:           messageUnpinnedHandler,
	event.MessageStamped:            messageStampedHandler,
	event.MessageUnstamped:          messageUnstampedHandler,
	event.ChannelCreated:            channelCreatedHandler,
	event.ChannelUpdated:            channelUpdatedHandler,
	event.ChannelDeleted:            channelDeletedHandler,
	event.ChannelStared:             channelStaredHandler,
	event.ChannelUnstared:           channelUnstaredHandler,
	event.ChannelRead:               channelReadHandler,
	event.ChannelViewersChanged:     channelViewersChangedHandler,
	event.ChannelSubscribersChanged: channelSubscribersChangedHandler,
	event.UserCreated:               userCreatedHandler,
	event.UserUpdated:               userUpdatedHandler,
	event.UserIconUpdated:           userIconUpdatedHandler,
	event.UserViewStateChanged:      userViewStateChangedHandler,
	event.UserTagAdded:              userTagUpdatedHandler,
	event.UserTagRemoved:            userTagUpdatedHandler,
	event.UserTagUpdated:            userTagUpdatedHandler,
	event.UserGroupCreated:          userGroupCreatedHandler,
	event.UserGroupUpdated:          userGroupUpdatedHandler,
	event.UserGroupDeleted:          userGroupDeletedHandler,
	event.UserGroupMemberAdded:      userGroupUpdatedHandler,
	event.UserGroupMemberUpdated:    userGroupUpdatedHandler,
	event.UserGroupMemberRemoved:    userGroupUpdatedHandler,
	event.UserGroupAdminAdded:       userGroupUpdatedHandler,
	event.UserGroupAdminRemoved:     userGroupUpdatedHandler,
	event.StampCreated:              stampCreatedHandler,
	event.StampUpdated:              stampUpdatedHandler,
	event.StampDeleted:              stampDeletedHandler,
	event.StampPaletteCreated:       stampPaletteCreatedHandler,
	event.StampPaletteUpdated:       stampPaletteUpdatedHandler,
	event.StampPaletteDeleted:       stampPaletteDeletedHandler,
	event.UserWebRTCv3StateChanged:  userWebRTCv3StateChangedHandler,
	event.ClipFolderCreated:         clipFolderCreatedHandler,
	event.ClipFolderUpdated:         clipFolderUpdatedHandler,
	event.ClipFolderDeleted:         clipFolderDeletedHandler,
	event.ClipFolderMessageDeleted:  clipFolderMessageDeletedHandler,
	event.ClipFolderMessageAdded:    clipFolderMessageAddedHandler,
	event.QallRoomStateChanged:      qallRoomStateChangedHandler,
	event.QallSoundboardItemCreated: qallSoundboardItemCreatedHandler,
	event.QallSoundboardItemDeleted: qallSoundboardItemDeletedHandler,
}

func (ns *Service) startConcurrentNotifications() {
	topics := make([]string, 0, len(handlerMap))
	for k := range handlerMap {
		topics = append(topics, k)
	}

	sub := ns.hub.Subscribe(notificationBufferSize, topics...)
	go func() {
		for msg := range sub.Receiver {
			h, ok := handlerMap[msg.Topic()]
			if ok {
				go h(ns, msg)
			}
		}
	}()
}

func messageCreatedHandler(ns *Service, ev hub.Message) {
	m := ev.Fields["message"].(*model.Message)
	parsed := ev.Fields["parse_result"].(*message.ParseResult)
	logger := ns.logger.With(zap.Stringer("messageId", m.ID))

	chTree := ns.cm.PublicChannelTree(context.Background())
	chID := m.ChannelID
	isDM := !chTree.IsChannelPresent(chID)
	forceNotify := chTree.IsForceChannel(chID)

	// 投稿ユーザー情報を取得
	mUser, err := ns.repo.GetUser(context.Background(), m.UserID, false)
	if err != nil {
		logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("userId", m.UserID)) // 失敗
		return
	}

	fcmPayload := &fcm.Payload{
		Type: "new_message",
		Icon: fmt.Sprintf("%s/api/v3/public/icon/%s", ns.origin, strings.ReplaceAll(mUser.GetName(), "#", "%23")),
		Tag:  "c:" + m.ChannelID.String(),
	}

	viewers := set.UUID{}       // バックグラウンドを含む対象チャンネル閲覧中のユーザー
	notifiedUsers := set.UUID{} // チャンネル通知購読ユーザー
	markedUsers := set.UUID{}   // チャンネル未読管理ユーザー
	noticeable := set.UUID{}    // noticeableな未読追加対象のユーザー
	citedUsers := set.UUID{}    // メッセージで引用されたメッセージを投稿したユーザー
	dmMembers := set.UUID{}     // isDMの場合 DMのメンバー

	// メッセージボディ作成
	if !isDM {
		// 公開チャンネル
		path := chTree.GetChannelPath(chID)
		fcmPayload.Title = "#" + path
		fcmPayload.Path = "/channels/" + path
		fcmPayload.SetBodyWithEllipsis(mUser.GetResponseDisplayName() + ": " + parsed.NotificationText())
	} else {
		// DM
		fcmPayload.Title = "@" + mUser.GetResponseDisplayName()
		fcmPayload.Path = "/users/" + mUser.GetName()
		fcmPayload.SetBodyWithEllipsis(parsed.NotificationText())
	}

	if len(parsed.Attachments) > 0 {
		if f, _ := ns.fm.Get(context.Background(), parsed.Attachments[0]); f != nil {
			if ok, _ := f.GetThumbnail(model.ThumbnailTypeImage); ok {
				fcmPayload.Image = optional.From(fmt.Sprintf("%s/api/v3/files/%s/thumbnail", ns.origin, f.GetID()))
			}
		}
	}

	// 対象者計算
	q := repository.UsersQuery{}.Active().NotBot()
	switch {
	case forceNotify: // 強制通知チャンネル
		users, err := ns.repo.GetUserIDs(context.Background(), q)
		if err != nil {
			logger.Error("failed to GetUsers", zap.Error(err)) // 失敗
			return
		}
		notifiedUsers.Add(users...)
		markedUsers.Add(users...)
		noticeable.Add(users...)

	case isDM: // DM
		users, err := ns.repo.GetUserIDs(context.Background(), q.CMemberOf(chID))
		if err != nil {
			logger.Error("failed to GetPrivateChannelMemberIDs", zap.Error(err), zap.Stringer("channelId", m.ChannelID)) // 失敗
			return
		}
		notifiedUsers.Add(users...)
		markedUsers.Add(users...)
		dmMembers.Add(users...)

	default: // 通常チャンネルメッセージ
		// チャンネル通知購読者取得
		notify, err := ns.repo.GetUserIDs(context.Background(), q.SubscriberAtNotifyLevelOf(chID))
		if err != nil {
			logger.Error("failed to GetUserIDs", zap.Error(err), zap.Stringer("channelId", m.ChannelID)) // 失敗
			return
		}
		notifiedUsers.Add(notify...)

		// チャンネル未読管理購読者取得
		mark, err := ns.repo.GetUserIDs(context.Background(), q.SubscriberAtMarkLevelOf(chID))
		if err != nil {
			logger.Error("failed to GetUserIDs", zap.Error(err), zap.Stringer("channelId", m.ChannelID)) // 失敗
			return
		}
		markedUsers.Add(mark...)

		// ユーザーグループ・メンションユーザー取得
		for _, uid := range parsed.Mentions {
			user, err := ns.repo.GetUser(context.Background(), uid, false)
			if err != nil {
				logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("userId", uid)) // 失敗
				continue
			}
			// 凍結ユーザー / Botの除外
			if !user.IsActive() || user.IsBot() {
				continue
			}

			notifiedUsers.Add(uid)
			markedUsers.Add(uid)
			noticeable.Add(uid)
		}
		for _, gid := range parsed.GroupMentions {
			gs, err := ns.repo.GetUserIDs(context.Background(), q.GMemberOf(gid))
			if err != nil {
				logger.Error("failed to GetUserGroupMemberIDs", zap.Error(err), zap.Stringer("groupId", gid)) // 失敗
				return
			}
			notifiedUsers.Add(gs...)
			markedUsers.Add(gs...)
			noticeable.Add(gs...)
		}
		// メッセージを引用されたユーザーへの通知
		for _, mid := range parsed.Citation {
			m, err := ns.repo.GetMessageByID(context.Background(), mid)
			if err != nil {
				logger.Error("failed to GetMessageByID", zap.Error(err), zap.Stringer("citedMessageId", mid)) // 失敗
				continue
			}
			uid := m.UserID

			user, err := ns.repo.GetUser(context.Background(), uid, false)
			if err != nil {
				logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("userId", uid)) // 失敗
				continue
			}
			// 凍結ユーザー / Botの除外
			if !user.IsActive() || user.IsBot() {
				continue
			}

			us, err := ns.repo.GetNotifyCitation(context.Background(), uid)
			if err != nil {
				logger.Error("failed to GetNotifyCitation", zap.Error(err), zap.Stringer("userId", uid)) // 失敗
				continue
			}

			markedUsers.Add(uid)
			noticeable.Add(uid)
			citedUsers.Add(uid)
			// 引用通知が有効の場合
			if us {
				notifiedUsers.Add(uid)
			}
		}
	}

	// チャンネル閲覧者取得
	for uid, swt := range ns.vm.GetChannelViewers(m.ChannelID) {
		viewers.Add(uid)
		if swt.State >= viewer.StateMonitoring {
			markedUsers.Remove(uid)   // 閲覧中ユーザーは未読管理から外す
			notifiedUsers.Remove(uid) // 閲覧中ユーザーは通知から外す
		}
	}

	// 未読追加
	markedUsers.Remove(m.UserID)

	userNoticeableMap := map[uuid.UUID]bool{}
	for uid := range markedUsers {
		userNoticeableMap[uid] = noticeable.Contains(uid)
	}
	if err := ns.repo.SetMessageUnreads(context.Background(), userNoticeableMap, m.ID); err != nil {
		logger.Error("failed to SetMessageUnreads", zap.Error(err), zap.Stringer("message_id", m.ID)) // 失敗
	}

	// WS送信
	var targetFuncNotCited ws.TargetFunc
	var targetFuncCited ws.TargetFunc
	if isDM {
		targetFuncNotCited = ws.TargetUserSets(dmMembers)
		targetFuncCited = ws.TargetNone()
	} else {
		targetFuncNotCited = ws.And(
			ws.Or(
				ws.TargetUserSets(markedUsers, viewers),
				ws.TargetTimelineStreamingEnabled(),
			),
			ws.Not(ws.TargetUserSets(citedUsers)),
		)
		targetFuncCited = ws.TargetUserSets(citedUsers)
	}
	go ns.messageWriter.WriteTypedMessage(ws.MessageCreated{ID: m.ID}, targetFuncNotCited)
	go ns.messageWriter.WriteTypedMessage(ws.MessageCreated{ID: m.ID, IsCiting: true}, targetFuncCited)

	// FCM送信
	targets := notifiedUsers.Clone()
	targets.Remove(m.UserID)
	ns.fcm.Send(targets, fcmPayload, true)
}

func messageUpdatedHandler(ns *Service, ev hub.Message) {
	cid := ev.Fields["message"].(*model.Message).ChannelID

	var targetFunc ws.TargetFunc
	if ns.cm.IsPublicChannel(context.Background(), cid) {
		// 公開チャンネル
		targetFunc = ws.Or(
			ws.TargetChannelViewers(cid),
			ws.TargetTimelineStreamingEnabled(),
		)
	} else {
		// DM
		targetFunc = ws.TargetChannelViewers(cid)
	}

	go ns.messageWriter.WriteTypedMessage(ws.MessageUpdated{ID: ev.Fields["message_id"].(uuid.UUID)}, targetFunc)
}

func messageDeletedHandler(ns *Service, ev hub.Message) {
	cid := ev.Fields["message"].(*model.Message).ChannelID

	var targetFunc ws.TargetFunc
	if ns.cm.IsPublicChannel(context.Background(), cid) {
		// 公開チャンネル
		targetFunc = ws.Or(
			ws.TargetChannelViewers(cid),
			ws.TargetTimelineStreamingEnabled(),
		)
	} else {
		// DM
		targetFunc = ws.TargetChannelViewers(cid)
	}

	go ns.messageWriter.WriteTypedMessage(ws.MessageDeleted{ID: ev.Fields["message_id"].(uuid.UUID)}, targetFunc)
}

func messagePinnedHandler(ns *Service, ev hub.Message) {
	channelViewerMulticast(ns, ev.Fields["channel_id"].(uuid.UUID),
		ws.MessagePinned{
			MessageID: ev.Fields["message_id"].(uuid.UUID),
			ChannelID: ev.Fields["channel_id"].(uuid.UUID),
		},
	)
}

func messageUnpinnedHandler(ns *Service, ev hub.Message) {
	channelViewerMulticast(ns, ev.Fields["channel_id"].(uuid.UUID),
		ws.MessageUnpinned{
			MessageID: ev.Fields["message_id"].(uuid.UUID),
			ChannelID: ev.Fields["channel_id"].(uuid.UUID),
		},
	)
}

func messageStampedHandler(ns *Service, ev hub.Message) {
	messageViewerMulticast(ns, ev.Fields["message_id"].(uuid.UUID),
		ws.MessageStamped{
			MessageID: ev.Fields["message_id"].(uuid.UUID),
			UserID:    ev.Fields["user_id"].(uuid.UUID),
			StampID:   ev.Fields["stamp_id"].(uuid.UUID),
			Count:     ev.Fields["count"].(int),
			CreatedAt: ev.Fields["created_at"].(time.Time),
		},
	)
}

func messageUnstampedHandler(ns *Service, ev hub.Message) {
	messageViewerMulticast(ns, ev.Fields["message_id"].(uuid.UUID),
		ws.MessageUnstamped{
			MessageID: ev.Fields["message_id"].(uuid.UUID),
			UserID:    ev.Fields["user_id"].(uuid.UUID),
			StampID:   ev.Fields["stamp_id"].(uuid.UUID),
		},
	)
}

func channelCreatedHandler(ns *Service, ev hub.Message) {
	channelHandler(ns, ev, func(body ws.ChannelBody) ws.TypedMessage { return ws.ChannelCreated(body) })
}

func channelUpdatedHandler(ns *Service, ev hub.Message) {
	channelHandler(ns, ev, func(body ws.ChannelBody) ws.TypedMessage { return ws.ChannelUpdated(body) })
}

func channelDeletedHandler(ns *Service, ev hub.Message) {
	channelHandler(ns, ev, func(body ws.ChannelBody) ws.TypedMessage { return ws.ChannelDeleted(body) })
}

func channelStaredHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		ws.ChannelStared{ID: ev.Fields["channel_id"].(uuid.UUID)},
	)
}

func channelUnstaredHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		ws.ChannelUnstared{ID: ev.Fields["channel_id"].(uuid.UUID)},
	)
}

func channelReadHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		ws.MessageRead{ID: ev.Fields["channel_id"].(uuid.UUID)},
	)
}

func channelViewersChangedHandler(ns *Service, ev hub.Message) {
	cid := ev.Fields["channel_id"].(uuid.UUID)
	viewers := viewer.ConvertToArray(ev.Fields["viewers"].(map[uuid.UUID]viewer.StateWithTime))
	payload := make([]ws.ChannelViewer, len(viewers))
	for i, v := range viewers {
		payload[i] = ws.ChannelViewer{UserID: v.UserID, State: v.State.String(), UpdatedAt: v.UpdatedAt}
	}
	channelViewerMulticast(ns, cid, ws.ChannelViewersChanged{ID: cid, Viewers: payload})
}

func channelSubscribersChangedHandler(ns *Service, ev hub.Message) {
	cid := ev.Fields["channel_id"].(uuid.UUID)
	uids := ev.Fields["subscriber_ids"].([]uuid.UUID)
	ns.messageWriter.WriteTypedMessage(
		ws.ChannelSubscribersChanged{ID: cid},
		ws.Or(ws.TargetChannelViewers(cid), ws.TargetUsers(uids...)),
	)
}

func userCreatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		ws.UserJoined{ID: ev.Fields["user_id"].(uuid.UUID)},
	)
}

func userUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		ws.UserUpdated{ID: ev.Fields["user_id"].(uuid.UUID)},
	)
}

func userIconUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		ws.UserIconUpdated{ID: ev.Fields["user_id"].(uuid.UUID)},
	)
}

func userViewStateChangedHandler(ns *Service, ev hub.Message) {
	states := ev.Fields["view_states"].(map[string]viewer.StateWithChannel)
	viewStates := make([]ws.MyChannelViewState, 0, len(states))
	for connKey, state := range states {
		viewStates = append(viewStates, ws.MyChannelViewState{
			Key:       connKey,
			ChannelID: state.ChannelID,
			State:     state.State.String(),
		})
	}

	uid := ev.Fields["user_id"].(uuid.UUID)
	userMulticast(ns, uid, ws.UserViewStateChanged{ViewStates: viewStates})
}

func userTagUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		ws.UserTagsUpdated{
			ID:    ev.Fields["user_id"].(uuid.UUID),
			TagID: ev.Fields["tag_id"].(uuid.UUID),
		},
	)
}

func userGroupCreatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		ws.UserGroupCreated{ID: ev.Fields["group_id"].(uuid.UUID)},
	)
}

func userGroupUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		ws.UserGroupUpdated{ID: ev.Fields["group_id"].(uuid.UUID)},
	)
}

func userGroupDeletedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		ws.UserGroupDeleted{ID: ev.Fields["group_id"].(uuid.UUID)},
	)
}

func stampCreatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		ws.StampCreated{ID: ev.Fields["stamp_id"].(uuid.UUID)},
	)
}

func stampUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		ws.StampUpdated{ID: ev.Fields["stamp_id"].(uuid.UUID)},
	)
}

func stampDeletedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		ws.StampDeleted{ID: ev.Fields["stamp_id"].(uuid.UUID)},
	)
}

func stampPaletteCreatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		ws.StampPaletteCreated{ID: ev.Fields["stamp_palette_id"].(uuid.UUID)},
	)
}

func stampPaletteUpdatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		ws.StampPaletteUpdated{ID: ev.Fields["stamp_palette_id"].(uuid.UUID)},
	)
}

func stampPaletteDeletedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		ws.StampPaletteDeleted{ID: ev.Fields["stamp_palette_id"].(uuid.UUID)},
	)
}

func userWebRTCv3StateChangedHandler(ns *Service, ev hub.Message) {
	states := ev.Fields["sessions"].(map[string]string)
	sessions := make([]ws.WebRTCSession, 0, len(states))
	for session, state := range states {
		sessions = append(sessions, ws.WebRTCSession{State: state, SessionID: session})
	}

	broadcast(ns, ws.UserWebRTCStateChanged{
		UserID:    ev.Fields["user_id"].(uuid.UUID),
		ChannelID: ev.Fields["channel_id"].(uuid.UUID),
		Sessions:  sessions,
	})
}

func qallRoomStateChangedHandler(ns *Service, ev hub.Message) {
	type roomState = ws.QallRoomStateChangedEventRoomStatesItem
	type participant = ws.QallRoomStateChangedEventRoomStatesItemParticipantsItem

	rooms := ev.Fields["roomStates"].([]qall.RoomWithParticipants)
	var states []roomState
	if rooms != nil {
		states = make([]roomState, len(rooms))
	}
	for i, room := range rooms {
		var participants []participant
		if room.Participants != nil {
			participants = make([]participant, len(room.Participants))
		}
		for j, p := range room.Participants {
			participants[j] = participant(p)
		}
		states[i] = roomState{
			RoomID:       room.RoomID,
			IsWebinar:    room.IsWebinar,
			Metadata:     room.Metadata,
			Participants: participants,
		}
	}
	broadcast(ns, ws.QallRoomStateChanged{RoomStates: states})
}

func qallSoundboardItemCreatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		ws.QallSoundboardItemCreated{
			SoundID:   ev.Fields["sound_id"].(uuid.UUID),
			Name:      ev.Fields["name"].(string),
			CreatorID: ev.Fields["creator_id"].(uuid.UUID),
		},
	)
}

func qallSoundboardItemDeletedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		ws.QallSoundboardItemDeleted{SoundID: ev.Fields["sound_id"].(uuid.UUID)},
	)
}

func clipFolderCreatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		ws.ClipFolderCreated{ID: ev.Fields["clip_folder_id"].(uuid.UUID)},
	)
}

func clipFolderUpdatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		ws.ClipFolderUpdated{ID: ev.Fields["clip_folder_id"].(uuid.UUID)},
	)
}

func clipFolderDeletedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		ws.ClipFolderDeleted{ID: ev.Fields["clip_folder_id"].(uuid.UUID)},
	)
}

func clipFolderMessageDeletedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		ws.ClipFolderMessageDeleted{
			FolderID:  ev.Fields["clip_folder_id"].(uuid.UUID),
			MessageID: ev.Fields["clip_folder_message_id"].(uuid.UUID),
		},
	)
}

func clipFolderMessageAddedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		ws.ClipFolderMessageAdded{
			FolderID:  ev.Fields["clip_folder_id"].(uuid.UUID),
			MessageID: ev.Fields["clip_folder_message_id"].(uuid.UUID),
		},
	)
}

func channelHandler(ns *Service, ev hub.Message, message func(ws.ChannelBody) ws.TypedMessage) {
	cid := ev.Fields["channel_id"].(uuid.UUID)
	private := ev.Fields["private"].(bool)
	if private {
		members, err := ns.cm.GetDMChannelMembers(context.Background(), cid)
		if err != nil {
			ns.logger.Error("failed to GetDMChannelMembers", zap.Error(err), zap.Stringer("channelId", cid))
			return
		}

		switch len(members) {
		case 1:
			go ns.messageWriter.WriteTypedMessage(message(ws.ChannelBody{ID: cid, DMUserID: new(members[0])}), ws.TargetUsers(members[0]))
		case 2:
			go ns.messageWriter.WriteTypedMessage(message(ws.ChannelBody{ID: cid, DMUserID: new(members[0])}), ws.TargetUsers(members[1]))
			go ns.messageWriter.WriteTypedMessage(message(ws.ChannelBody{ID: cid, DMUserID: new(members[1])}), ws.TargetUsers(members[0]))
		default:
			ns.logger.Error("private channel event not defined", zap.Stringer("cid", cid))
		}
	} else {
		go ns.messageWriter.WriteTypedMessage(message(ws.ChannelBody{ID: cid}), ws.TargetAll())
	}
}

func channelViewerMulticast(ns *Service, cid uuid.UUID, msg ws.TypedMessage) {
	go ns.messageWriter.WriteTypedMessage(msg, ws.TargetChannelViewers(cid))
}

func messageViewerMulticast(ns *Service, mid uuid.UUID, msg ws.TypedMessage) {
	m, err := ns.mm.Get(context.Background(), mid)
	if err != nil {
		ns.logger.Error("failed to GetMessageByID", zap.Error(err), zap.Stringer("messageId", mid)) // 失敗
		return
	}
	channelViewerMulticast(ns, m.GetChannelID(), msg)
}

func broadcast(ns *Service, msg ws.TypedMessage) {
	go ns.messageWriter.WriteTypedMessage(msg, ws.TargetAll())
}

func userMulticast(ns *Service, userID uuid.UUID, msg ws.TypedMessage) {
	go ns.messageWriter.WriteTypedMessage(msg, ws.TargetUsers(userID))
}
