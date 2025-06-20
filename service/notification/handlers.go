package notification

import (
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
	event.UserOnline:                userOnlineHandler,
	event.UserOffline:               userOfflineHandler,
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

func messageCreatedHandler(ns *Service, ev hub.Message) {
	m := ev.Fields["message"].(*model.Message)
	parsed := ev.Fields["parse_result"].(*message.ParseResult)
	logger := ns.logger.With(zap.Stringer("messageId", m.ID))

	chTree := ns.cm.PublicChannelTree()
	chID := m.ChannelID
	isDM := !chTree.IsChannelPresent(chID)
	forceNotify := chTree.IsForceChannel(chID)

	// 投稿ユーザー情報を取得
	mUser, err := ns.repo.GetUser(m.UserID, false)
	if err != nil {
		logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("userId", m.UserID)) // 失敗
		return
	}

	fcmPayload := &fcm.Payload{
		Type: "new_message",
		Icon: fmt.Sprintf("%s/api/v3/public/icon/%s", ns.origin, strings.ReplaceAll(mUser.GetName(), "#", "%23")),
		Tag:  "c:" + m.ChannelID.String(),
	}
	wsEventType := "MESSAGE_CREATED"
	wsPayloadNotCited := map[string]interface{}{
		"id":        m.ID,
		"is_citing": false,
	}
	wsPayloadCited := map[string]interface{}{
		"id":        m.ID,
		"is_citing": true,
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
		if f, _ := ns.fm.Get(parsed.Attachments[0]); f != nil {
			if ok, _ := f.GetThumbnail(model.ThumbnailTypeImage); ok {
				fcmPayload.Image = optional.From(fmt.Sprintf("%s/api/v3/files/%s/thumbnail", ns.origin, f.GetID()))
			}
		}
	}

	// 対象者計算
	q := repository.UsersQuery{}.Active().NotBot()
	switch {
	case forceNotify: // 強制通知チャンネル
		users, err := ns.repo.GetUserIDs(q)
		if err != nil {
			logger.Error("failed to GetUsers", zap.Error(err)) // 失敗
			return
		}
		notifiedUsers.Add(users...)
		markedUsers.Add(users...)
		noticeable.Add(users...)

	case isDM: // DM
		users, err := ns.repo.GetUserIDs(q.CMemberOf(chID))
		if err != nil {
			logger.Error("failed to GetPrivateChannelMemberIDs", zap.Error(err), zap.Stringer("channelId", m.ChannelID)) // 失敗
			return
		}
		notifiedUsers.Add(users...)
		markedUsers.Add(users...)
		dmMembers.Add(users...)

	default: // 通常チャンネルメッセージ
		// チャンネル通知購読者取得
		notify, err := ns.repo.GetUserIDs(q.SubscriberAtNotifyLevelOf(chID))
		if err != nil {
			logger.Error("failed to GetUserIDs", zap.Error(err), zap.Stringer("channelId", m.ChannelID)) // 失敗
			return
		}
		notifiedUsers.Add(notify...)

		// チャンネル未読管理購読者取得
		mark, err := ns.repo.GetUserIDs(q.SubscriberAtMarkLevelOf(chID))
		if err != nil {
			logger.Error("failed to GetUserIDs", zap.Error(err), zap.Stringer("channelId", m.ChannelID)) // 失敗
			return
		}
		markedUsers.Add(mark...)

		// ユーザーグループ・メンションユーザー取得
		for _, uid := range parsed.Mentions {
			user, err := ns.repo.GetUser(uid, false)
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
			gs, err := ns.repo.GetUserIDs(q.GMemberOf(gid))
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
			m, err := ns.repo.GetMessageByID(mid)
			if err != nil {
				logger.Error("failed to GetMessageByID", zap.Error(err), zap.Stringer("citedMessageId", mid)) // 失敗
				continue
			}
			uid := m.UserID

			user, err := ns.repo.GetUser(uid, false)
			if err != nil {
				logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("userId", uid)) // 失敗
				continue
			}
			// 凍結ユーザー / Botの除外
			if !user.IsActive() || user.IsBot() {
				continue
			}

			us, err := ns.repo.GetNotifyCitation(uid)
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
		if swt.State > viewer.StateNone {
			markedUsers.Remove(uid)   // 閲覧中ユーザーは未読管理から外す
			notifiedUsers.Remove(uid) // 閲覧中ユーザーは通知から外す
		}
	}

	// 未読追加
	markedUsers.Remove(m.UserID)

	userNoticeableMap := map[uuid.UUID]bool{}
	for uid := range markedUsers {
		if noticeable.Contains(uid) {
			userNoticeableMap[uid] = true
		} else {
			userNoticeableMap[uid] = false
		}
	}
	if err := ns.repo.SetMessageUnreads(userNoticeableMap, m.ID); err != nil {
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
	go ns.ws.WriteMessage(wsEventType, wsPayloadNotCited, targetFuncNotCited)
	go ns.ws.WriteMessage(wsEventType, wsPayloadCited, targetFuncCited)

	// FCM送信
	targets := notifiedUsers.Clone()
	targets.Remove(m.UserID)
	ns.fcm.Send(targets, fcmPayload, true)
}

func messageUpdatedHandler(ns *Service, ev hub.Message) {
	cid := ev.Fields["message"].(*model.Message).ChannelID
	wsEventType := "MESSAGE_UPDATED"
	wsPayload := map[string]interface{}{
		"id": ev.Fields["message_id"].(uuid.UUID),
	}

	var targetFunc ws.TargetFunc
	if ns.cm.IsPublicChannel(cid) {
		// 公開チャンネル
		targetFunc = ws.Or(
			ws.TargetChannelViewers(cid),
			ws.TargetTimelineStreamingEnabled(),
		)
	} else {
		// DM
		targetFunc = ws.TargetChannelViewers(cid)
	}

	go ns.ws.WriteMessage(wsEventType, wsPayload, targetFunc)
}

func messageDeletedHandler(ns *Service, ev hub.Message) {
	cid := ev.Fields["message"].(*model.Message).ChannelID
	wsEventType := "MESSAGE_DELETED"
	wsPayload := map[string]interface{}{
		"id": ev.Fields["message_id"].(uuid.UUID),
	}

	var targetFunc ws.TargetFunc
	if ns.cm.IsPublicChannel(cid) {
		// 公開チャンネル
		targetFunc = ws.Or(
			ws.TargetChannelViewers(cid),
			ws.TargetTimelineStreamingEnabled(),
		)
	} else {
		// DM
		targetFunc = ws.TargetChannelViewers(cid)
	}

	go ns.ws.WriteMessage(wsEventType, wsPayload, targetFunc)
}

func messagePinnedHandler(ns *Service, ev hub.Message) {
	channelViewerMulticast(ns, ev.Fields["channel_id"].(uuid.UUID),
		"MESSAGE_PINNED",
		map[string]interface{}{
			"message_id": ev.Fields["message_id"].(uuid.UUID),
			"channel_id": ev.Fields["channel_id"].(uuid.UUID),
		},
	)
}

func messageUnpinnedHandler(ns *Service, ev hub.Message) {
	channelViewerMulticast(ns, ev.Fields["channel_id"].(uuid.UUID),
		"MESSAGE_UNPINNED",
		map[string]interface{}{
			"message_id": ev.Fields["message_id"].(uuid.UUID),
			"channel_id": ev.Fields["channel_id"].(uuid.UUID),
		},
	)
}

func messageStampedHandler(ns *Service, ev hub.Message) {
	messageViewerMulticast(ns, ev.Fields["message_id"].(uuid.UUID),
		"MESSAGE_STAMPED",
		map[string]interface{}{
			"message_id": ev.Fields["message_id"].(uuid.UUID),
			"user_id":    ev.Fields["user_id"].(uuid.UUID),
			"stamp_id":   ev.Fields["stamp_id"].(uuid.UUID),
			"count":      ev.Fields["count"].(int),
			"created_at": ev.Fields["created_at"].(time.Time),
		},
	)
}

func messageUnstampedHandler(ns *Service, ev hub.Message) {
	messageViewerMulticast(ns, ev.Fields["message_id"].(uuid.UUID),
		"MESSAGE_UNSTAMPED",
		map[string]interface{}{
			"message_id": ev.Fields["message_id"].(uuid.UUID),
			"user_id":    ev.Fields["user_id"].(uuid.UUID),
			"stamp_id":   ev.Fields["stamp_id"].(uuid.UUID),
		},
	)
}

func channelCreatedHandler(ns *Service, ev hub.Message) {
	channelHandler(ns, ev, "CHANNEL_CREATED")
}

func channelUpdatedHandler(ns *Service, ev hub.Message) {
	channelHandler(ns, ev, "CHANNEL_UPDATED")
}

func channelDeletedHandler(ns *Service, ev hub.Message) {
	channelHandler(ns, ev, "CHANNEL_DELETED")
}

func channelStaredHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		"CHANNEL_STARED",
		map[string]interface{}{
			"id": ev.Fields["channel_id"].(uuid.UUID),
		},
	)
}

func channelUnstaredHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		"CHANNEL_UNSTARED",
		map[string]interface{}{
			"id": ev.Fields["channel_id"].(uuid.UUID),
		},
	)
}

func channelReadHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		"MESSAGE_READ",
		map[string]interface{}{
			"id": ev.Fields["channel_id"].(uuid.UUID),
		},
	)
}

func channelViewersChangedHandler(ns *Service, ev hub.Message) {
	cid := ev.Fields["channel_id"].(uuid.UUID)
	channelViewerMulticast(ns, cid,
		"CHANNEL_VIEWERS_CHANGED",
		map[string]interface{}{
			"id":      cid,
			"viewers": viewer.ConvertToArray(ev.Fields["viewers"].(map[uuid.UUID]viewer.StateWithTime)),
		},
	)
}

func channelSubscribersChangedHandler(ns *Service, ev hub.Message) {
	cid := ev.Fields["channel_id"].(uuid.UUID)
	uids := ev.Fields["subscriber_ids"].([]uuid.UUID)
	ns.ws.WriteMessage(
		"CHANNEL_SUBSCRIBERS_CHANGED",
		map[string]interface{}{
			"id": cid,
		},
		ws.Or(ws.TargetChannelViewers(cid), ws.TargetUsers(uids...)),
	)
}

func userCreatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"USER_JOINED",
		map[string]interface{}{
			"id": ev.Fields["user_id"].(uuid.UUID),
		},
	)
}

func userUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"USER_UPDATED",
		map[string]interface{}{
			"id": ev.Fields["user_id"].(uuid.UUID),
		},
	)
}

func userIconUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"USER_ICON_UPDATED",
		map[string]interface{}{
			"id": ev.Fields["user_id"].(uuid.UUID),
		},
	)
}

func userOnlineHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"USER_ONLINE",
		map[string]interface{}{
			"id": ev.Fields["user_id"].(uuid.UUID),
		},
	)
}

func userOfflineHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"USER_OFFLINE",
		map[string]interface{}{
			"id": ev.Fields["user_id"].(uuid.UUID),
		},
	)
}

func userViewStateChangedHandler(ns *Service, ev hub.Message) {
	type ViewState struct {
		Key       string    `json:"key"`
		ChannelID uuid.UUID `json:"channelId"`
		State     string    `json:"state"`
	}
	viewStates := make([]ViewState, 0)
	for connKey, state := range ev.Fields["view_states"].(map[string]viewer.StateWithChannel) {
		viewStates = append(viewStates, ViewState{
			Key:       connKey,
			ChannelID: state.ChannelID,
			State:     state.State.String(),
		})
	}

	uid := ev.Fields["user_id"].(uuid.UUID)
	userMulticast(ns, uid,
		"USER_VIEWSTATE_CHANGED",
		map[string]interface{}{
			"view_states": viewStates,
		},
	)
}

func userTagUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"USER_TAGS_UPDATED",
		map[string]interface{}{
			"id":     ev.Fields["user_id"].(uuid.UUID),
			"tag_id": ev.Fields["tag_id"].(uuid.UUID),
		},
	)
}

func userGroupCreatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"USER_GROUP_CREATED",
		map[string]interface{}{
			"id": ev.Fields["group_id"].(uuid.UUID),
		},
	)
}

func userGroupUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"USER_GROUP_UPDATED",
		map[string]interface{}{
			"id": ev.Fields["group_id"].(uuid.UUID),
		},
	)
}

func userGroupDeletedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"USER_GROUP_DELETED",
		map[string]interface{}{
			"id": ev.Fields["group_id"].(uuid.UUID),
		},
	)
}

func stampCreatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"STAMP_CREATED",
		map[string]interface{}{
			"id": ev.Fields["stamp_id"].(uuid.UUID),
		},
	)
}

func stampUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"STAMP_UPDATED",
		map[string]interface{}{
			"id": ev.Fields["stamp_id"].(uuid.UUID),
		},
	)
}

func stampDeletedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"STAMP_DELETED",
		map[string]interface{}{
			"id": ev.Fields["stamp_id"].(uuid.UUID),
		},
	)
}

func stampPaletteCreatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		"STAMP_PALETTE_CREATED",
		map[string]interface{}{
			"id": ev.Fields["stamp_palette_id"].(uuid.UUID),
		},
	)
}

func stampPaletteUpdatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		"STAMP_PALETTE_UPDATED",
		map[string]interface{}{
			"id": ev.Fields["stamp_palette_id"].(uuid.UUID),
		},
	)
}

func stampPaletteDeletedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		"STAMP_PALETTE_DELETED",
		map[string]interface{}{
			"id": ev.Fields["stamp_palette_id"].(uuid.UUID),
		},
	)
}

func userWebRTCv3StateChangedHandler(ns *Service, ev hub.Message) {
	type StateSession struct {
		State     string `json:"state"`
		SessionID string `json:"sessionId"`
	}
	sessions := make([]StateSession, 0)
	for session, state := range ev.Fields["sessions"].(map[string]string) {
		sessions = append(sessions, StateSession{State: state, SessionID: session})
	}

	broadcast(ns,
		"USER_WEBRTC_STATE_CHANGED",
		map[string]interface{}{
			"user_id":    ev.Fields["user_id"].(uuid.UUID),
			"channel_id": ev.Fields["channel_id"].(uuid.UUID),
			"sessions":   sessions,
		},
	)
}

func qallRoomStateChangedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"QALL_ROOM_STATE_CHANGED",
		map[string]interface{}{
			"roomStates": ev.Fields["roomStates"].([]qall.RoomWithParticipants),
		},
	)
}

func qallSoundboardItemCreatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"QALL_SOUNDBOARD_ITEM_CREATED",
		map[string]interface{}{
			"sound_id":   ev.Fields["sound_id"].(uuid.UUID),
			"name":       ev.Fields["name"].(string),
			"creator_id": ev.Fields["creator_id"].(uuid.UUID),
		},
	)
}

func qallSoundboardItemDeletedHandler(ns *Service, ev hub.Message) {
	broadcast(ns,
		"QALL_SOUNDBOARD_ITEM_DELETED",
		map[string]interface{}{
			"sound_id": ev.Fields["sound_id"].(uuid.UUID),
		},
	)
}

func clipFolderCreatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		"CLIP_FOLDER_CREATED",
		map[string]interface{}{
			"id": ev.Fields["clip_folder_id"].(uuid.UUID),
		},
	)
}

func clipFolderUpdatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		"CLIP_FOLDER_UPDATED",
		map[string]interface{}{
			"id": ev.Fields["clip_folder_id"].(uuid.UUID),
		},
	)
}

func clipFolderDeletedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		"CLIP_FOLDER_DELETED",
		map[string]interface{}{
			"id": ev.Fields["clip_folder_id"].(uuid.UUID),
		},
	)
}

func clipFolderMessageDeletedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		"CLIP_FOLDER_MESSAGE_DELETED",
		map[string]interface{}{
			"folder_id":  ev.Fields["clip_folder_id"].(uuid.UUID),
			"message_id": ev.Fields["clip_folder_message_id"].(uuid.UUID),
		},
	)
}

func clipFolderMessageAddedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID),
		"CLIP_FOLDER_MESSAGE_ADDED",
		map[string]interface{}{
			"folder_id":  ev.Fields["clip_folder_id"].(uuid.UUID),
			"message_id": ev.Fields["clip_folder_message_id"].(uuid.UUID),
		},
	)
}

func channelHandler(ns *Service, ev hub.Message, eventType string) {
	cid := ev.Fields["channel_id"].(uuid.UUID)
	private := ev.Fields["private"].(bool)
	if private {
		members, err := ns.cm.GetDMChannelMembers(cid)
		if err != nil {
			ns.logger.Error("failed to GetDMChannelMembers", zap.Error(err), zap.Stringer("channelId", cid))
			return
		}

		switch len(members) {
		case 1:
			go ns.ws.WriteMessage(eventType, map[string]interface{}{
				"id":         cid,
				"dm_user_id": members[0],
			}, ws.TargetUsers(members[0]))
		case 2:
			go ns.ws.WriteMessage(eventType, map[string]interface{}{
				"id":         cid,
				"dm_user_id": members[0],
			}, ws.TargetUsers(members[1]))
			go ns.ws.WriteMessage(eventType, map[string]interface{}{
				"id":         cid,
				"dm_user_id": members[1],
			}, ws.TargetUsers(members[0]))
		default:
			ns.logger.Error("private channel event not defined", zap.Stringer("cid", cid))
		}
	} else {
		go ns.ws.WriteMessage(eventType, map[string]interface{}{
			"id": cid,
		}, ws.TargetAll())
	}
}

func channelViewerMulticast(ns *Service, cid uuid.UUID, wsEventType string, wsPayload interface{}) {
	go ns.ws.WriteMessage(wsEventType, wsPayload, ws.TargetChannelViewers(cid))
}

func messageViewerMulticast(ns *Service, mid uuid.UUID, wsEventType string, wsPayload interface{}) {
	m, err := ns.mm.Get(mid)
	if err != nil {
		ns.logger.Error("failed to GetMessageByID", zap.Error(err), zap.Stringer("messageId", mid)) // 失敗
		return
	}
	channelViewerMulticast(ns, m.GetChannelID(), wsEventType, wsPayload)
}

func broadcast(ns *Service, wsEventType string, wsPayload interface{}) {
	go ns.ws.WriteMessage(wsEventType, wsPayload, ws.TargetAll())
}

func userMulticast(ns *Service, userID uuid.UUID, wsEventType string, wsPayload interface{}) {
	go ns.ws.WriteMessage(wsEventType, wsPayload, ws.TargetUsers(userID))
}
