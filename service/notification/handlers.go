package notification

import (
	"fmt"
	"strings"
	"time"

	"github.com/traPtitech/traQ/utils/optional"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/fcm"
	"github.com/traPtitech/traQ/service/sse"
	"github.com/traPtitech/traQ/service/viewer"
	"github.com/traPtitech/traQ/service/ws"
	"github.com/traPtitech/traQ/utils/message"
	"github.com/traPtitech/traQ/utils/set"
	"go.uber.org/zap"
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
	event.MessageCited:              messageCitedHandler,
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
	event.UserTagAdded:              userTagUpdatedHandler,
	event.UserTagRemoved:            userTagUpdatedHandler,
	event.UserTagUpdated:            userTagUpdatedHandler,
	event.UserGroupCreated:          userGroupCreatedHandler,
	event.UserGroupDeleted:          userGroupDeletedHandler,
	event.UserGroupMemberAdded:      userGroupUpdatedHandler,
	event.UserGroupMemberRemoved:    userGroupUpdatedHandler,
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
	ssePayload := &sse.EventData{
		EventType: "MESSAGE_CREATED",
		Payload: map[string]interface{}{
			"id": m.ID,
		},
	}

	viewers := set.UUID{}       // バックグラウンドを含む対象チャンネル閲覧中のユーザー
	notifiedUsers := set.UUID{} // チャンネル通知購読ユーザー
	markedUsers := set.UUID{}   // チャンネル未読管理ユーザー
	noticeable := set.UUID{}    // noticeableな未読追加対象のユーザー

	// メッセージボディ作成
	if !isDM {
		// 公開チャンネル
		path := chTree.GetChannelPath(chID)
		fcmPayload.Title = "#" + path
		fcmPayload.Path = "/channels/" + path
		fcmPayload.SetBodyWithEllipsis(mUser.GetResponseDisplayName() + ": " + parsed.OneLine())
	} else {
		// DM
		fcmPayload.Title = "@" + mUser.GetResponseDisplayName()
		fcmPayload.Path = "/users/" + mUser.GetName()
		fcmPayload.SetBodyWithEllipsis(parsed.OneLine())
	}

	if len(parsed.Attachments) > 0 {
		if f, _ := ns.fm.Get(parsed.Attachments[0]); f != nil && f.HasThumbnail() {
			fcmPayload.Image = optional.StringFrom(fmt.Sprintf("%s/api/v3/files/%s/thumbnail", ns.origin, f.GetID()))
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
			// 凍結ユーザーの除外
			if !user.IsActive() {
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
		//メッセージを引用されたユーザーへの通知
		for _, mid := range parsed.Citation {
			m, err := ns.repo.GetMessageByID(mid)
			if err != nil {
				logger.Error("failed to GetMessageByID", zap.Error(err), zap.Stringer("citedMessageId", mid)) // 失敗
				continue
			}
			uid := m.UserID
			us, err := ns.repo.GetNotifyCitation(uid)
			if err != nil {
				logger.Error("failed to GetNotifyCitation", zap.Error(err), zap.Stringer("userId", uid)) // 失敗
				continue
			}
			if us.NotifyCitation {
				notifiedUsers.Add(uid)
				markedUsers.Add(uid)
				noticeable.Add(uid)
			}
		}
	}

	// チャンネル閲覧者取得
	for uid, swt := range ns.vm.GetChannelViewers(m.ChannelID) {
		viewers.Add(uid)
		if swt.State > viewer.StateNone {
			markedUsers.Remove(uid) // 閲覧中ユーザーは未読管理から外す
		}
	}

	// 未読追加
	markedUsers.Remove(m.UserID)
	for id := range markedUsers {
		err := ns.repo.SetMessageUnread(id, m.ID, noticeable.Contains(id))
		if err != nil {
			logger.Error("failed to SetMessageUnread", zap.Error(err), zap.Stringer("user_id", id)) // 失敗
		}
	}

	// WS送信
	var targetFunc ws.TargetFunc
	if isDM {
		targetFunc = ws.TargetUserSets(notifiedUsers)
	} else {
		targetFunc = ws.Or(
			ws.TargetUserSets(notifiedUsers, viewers),
			ws.TargetTimelineStreamingEnabled(),
		)
	}
	go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, targetFunc)

	// FCM送信
	targets := notifiedUsers.Clone()
	targets.Remove(m.UserID)
	ns.fcm.Send(targets, fcmPayload, true)
}

func messageUpdatedHandler(ns *Service, ev hub.Message) {
	cid := ev.Fields["message"].(*model.Message).ChannelID
	ssePayload := &sse.EventData{
		EventType: "MESSAGE_UPDATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["message_id"].(uuid.UUID),
		},
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

	go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, targetFunc)
}

func messageDeletedHandler(ns *Service, ev hub.Message) {
	cid := ev.Fields["message"].(*model.Message).ChannelID
	ssePayload := &sse.EventData{
		EventType: "MESSAGE_DELETED",
		Payload: map[string]interface{}{
			"id": ev.Fields["message_id"].(uuid.UUID),
		},
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

	go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, targetFunc)
}

func messagePinnedHandler(ns *Service, ev hub.Message) {
	channelViewerMulticast(ns, ev.Fields["channel_id"].(uuid.UUID), &sse.EventData{
		EventType: "MESSAGE_PINNED",
		Payload: map[string]interface{}{
			"message_id": ev.Fields["message_id"].(uuid.UUID),
			"channel_id": ev.Fields["channel_id"].(uuid.UUID),
		},
	})
}

func messageUnpinnedHandler(ns *Service, ev hub.Message) {
	channelViewerMulticast(ns, ev.Fields["channel_id"].(uuid.UUID), &sse.EventData{
		EventType: "MESSAGE_UNPINNED",
		Payload: map[string]interface{}{
			"message_id": ev.Fields["message_id"].(uuid.UUID),
			"channel_id": ev.Fields["channel_id"].(uuid.UUID),
		},
	})
}

func messageStampedHandler(ns *Service, ev hub.Message) {
	messageViewerMulticast(ns, ev.Fields["message_id"].(uuid.UUID), &sse.EventData{
		EventType: "MESSAGE_STAMPED",
		Payload: map[string]interface{}{
			"message_id": ev.Fields["message_id"].(uuid.UUID),
			"user_id":    ev.Fields["user_id"].(uuid.UUID),
			"stamp_id":   ev.Fields["stamp_id"].(uuid.UUID),
			"count":      ev.Fields["count"].(int),
			"created_at": ev.Fields["created_at"].(time.Time),
		},
	})
}

func messageUnstampedHandler(ns *Service, ev hub.Message) {
	messageViewerMulticast(ns, ev.Fields["message_id"].(uuid.UUID), &sse.EventData{
		EventType: "MESSAGE_UNSTAMPED",
		Payload: map[string]interface{}{
			"message_id": ev.Fields["message_id"].(uuid.UUID),
			"user_id":    ev.Fields["user_id"].(uuid.UUID),
			"stamp_id":   ev.Fields["stamp_id"].(uuid.UUID),
		},
	})
}

func messageCitedHandler(ns *Service, ev hub.Message) {
	logger := ns.logger.With(zap.Stringer("messageId", ev.Fields["message_id"].(uuid.UUID)))
	for _, mid := range ev.Fields["cited_ids"].([]uuid.UUID) {
		m, err := ns.repo.GetMessageByID(mid)
		if err != nil {
			logger.Error("failed to GetMessageByID", zap.Error(err), zap.Stringer("citedMessageId", mid)) // 失敗
			continue
		}
		userMulticast(ns, m.UserID, &sse.EventData{
			EventType: "MESSAGE_CITED",
			Payload: map[string]interface{}{
				"message_id": ev.Fields["message_id"].(uuid.UUID),
				"channel_id": ev.Fields["message"].(*model.Message).ChannelID,
				"user_id":    ev.Fields["message"].(*model.Message).UserID,
			},
		})
	}

}

func channelCreatedHandler(ns *Service, ev hub.Message) {
	channelHandler(ns, ev, &sse.EventData{
		EventType: "CHANNEL_CREATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["channel_id"].(uuid.UUID),
		},
	})
}

func channelUpdatedHandler(ns *Service, ev hub.Message) {
	channelHandler(ns, ev, &sse.EventData{
		EventType: "CHANNEL_UPDATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["channel_id"].(uuid.UUID),
		},
	})
}

func channelDeletedHandler(ns *Service, ev hub.Message) {
	channelHandler(ns, ev, &sse.EventData{
		EventType: "CHANNEL_DELETED",
		Payload: map[string]interface{}{
			"id": ev.Fields["channel_id"].(uuid.UUID),
		},
	})
}

func channelStaredHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CHANNEL_STARED",
		Payload: map[string]interface{}{
			"id": ev.Fields["channel_id"].(uuid.UUID),
		},
	})
}

func channelUnstaredHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CHANNEL_UNSTARED",
		Payload: map[string]interface{}{
			"id": ev.Fields["channel_id"].(uuid.UUID),
		},
	})
}

func channelReadHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "MESSAGE_READ",
		Payload: map[string]interface{}{
			"id": ev.Fields["channel_id"].(uuid.UUID),
		},
	})
}

func channelViewersChangedHandler(ns *Service, ev hub.Message) {
	cid := ev.Fields["channel_id"].(uuid.UUID)
	channelViewerMulticast(ns, cid, &sse.EventData{
		EventType: "CHANNEL_VIEWERS_CHANGED",
		Payload: map[string]interface{}{
			"id":      cid,
			"viewers": viewer.ConvertToArray(ev.Fields["viewers"].(map[uuid.UUID]viewer.StateWithTime)),
		},
	})
}

func channelSubscribersChangedHandler(ns *Service, ev hub.Message) {
	cid := ev.Fields["channel_id"].(uuid.UUID)
	channelViewerMulticast(ns, cid, &sse.EventData{
		EventType: "CHANNEL_SUBSCRIBERS_CHANGED",
		Payload: map[string]interface{}{
			"id": cid,
		},
	})
}

func userCreatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_JOINED",
		Payload: map[string]interface{}{
			"id": ev.Fields["user_id"].(uuid.UUID),
		},
	})
}

func userUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_UPDATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["user_id"].(uuid.UUID),
		},
	})
}

func userIconUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_ICON_UPDATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["user_id"].(uuid.UUID),
		},
	})
}

func userOnlineHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_ONLINE",
		Payload: map[string]interface{}{
			"id": ev.Fields["user_id"].(uuid.UUID),
		},
	})
}

func userOfflineHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_OFFLINE",
		Payload: map[string]interface{}{
			"id": ev.Fields["user_id"].(uuid.UUID),
		},
	})
}

func userTagUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_TAGS_UPDATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["user_id"].(uuid.UUID),
		},
	})
}

func userGroupCreatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_GROUP_CREATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["group_id"].(uuid.UUID),
		},
	})
}

func userGroupUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_GROUP_UPDATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["group_id"].(uuid.UUID),
		},
	})
}

func userGroupDeletedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_GROUP_DELETED",
		Payload: map[string]interface{}{
			"id": ev.Fields["group_id"].(uuid.UUID),
		},
	})
}

func stampCreatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "STAMP_CREATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["stamp_id"].(uuid.UUID),
		},
	})
}

func stampUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "STAMP_UPDATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["stamp_id"].(uuid.UUID),
		},
	})
}

func stampDeletedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "STAMP_DELETED",
		Payload: map[string]interface{}{
			"id": ev.Fields["stamp_id"].(uuid.UUID),
		},
	})
}

func stampPaletteCreatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "STAMP_PALETTE_CREATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["stamp_palette_id"].(uuid.UUID),
		},
	})
}

func stampPaletteUpdatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "STAMP_PALETTE_UPDATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["stamp_palette_id"].(uuid.UUID),
		},
	})
}

func stampPaletteDeletedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "STAMP_PALETTE_DELETED",
		Payload: map[string]interface{}{
			"id": ev.Fields["stamp_palette_id"].(uuid.UUID),
		},
	})
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

	go ns.ws.WriteMessage("USER_WEBRTC_STATE_CHANGED", map[string]interface{}{
		"user_id":    ev.Fields["user_id"].(uuid.UUID),
		"channel_id": ev.Fields["channel_id"].(uuid.UUID),
		"sessions":   sessions,
	}, ws.TargetAll())
}

func clipFolderCreatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CLIP_FOLDER_CREATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["clip_folder_id"].(uuid.UUID),
		},
	})
}

func clipFolderUpdatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CLIP_FOLDER_UPDATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["clip_folder_id"].(uuid.UUID),
		},
	})
}

func clipFolderDeletedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CLIP_FOLDER_DELETED",
		Payload: map[string]interface{}{
			"id": ev.Fields["clip_folder_id"].(uuid.UUID),
		},
	})
}

func clipFolderMessageDeletedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CLIP_FOLDER_MESSAGE_DELETED",
		Payload: map[string]interface{}{
			"folder_id":  ev.Fields["clip_folder_id"].(uuid.UUID),
			"message_id": ev.Fields["clip_folder_message_id"].(uuid.UUID),
		},
	})
}

func clipFolderMessageAddedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CLIP_FOLDER_MESSAGE_ADDED",
		Payload: map[string]interface{}{
			"folder_id":  ev.Fields["clip_folder_id"].(uuid.UUID),
			"message_id": ev.Fields["clip_folder_message_id"].(uuid.UUID),
		},
	})
}

func channelHandler(ns *Service, ev hub.Message, ssePayload *sse.EventData) {
	private := ev.Fields["private"].(bool)
	if private {
		cid := ev.Fields["channel_id"].(uuid.UUID)
		members, err := ns.cm.GetDMChannelMembers(cid)
		if err != nil {
			ns.logger.Error("failed to GetDMChannelMembers", zap.Error(err), zap.Stringer("channelId", cid))
			return
		}
		go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, ws.TargetUsers(members...))
	} else {
		go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, ws.TargetAll())
	}
}

func channelViewerMulticast(ns *Service, cid uuid.UUID, ssePayload *sse.EventData) {
	go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, ws.TargetChannelViewers(cid))
}

func messageViewerMulticast(ns *Service, mid uuid.UUID, ssePayload *sse.EventData) {
	m, err := ns.mm.Get(mid)
	if err != nil {
		ns.logger.Error("failed to GetMessageByID", zap.Error(err), zap.Stringer("messageId", mid)) // 失敗
		return
	}
	channelViewerMulticast(ns, m.GetChannelID(), ssePayload)
}

func broadcast(ns *Service, ssePayload *sse.EventData) {
	go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, ws.TargetAll())
}

func userMulticast(ns *Service, userID uuid.UUID, ssePayload *sse.EventData) {
	go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, ws.TargetUsers(userID))
}
