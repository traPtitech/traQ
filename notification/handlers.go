package notification

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/notification/fcm"
	"github.com/traPtitech/traQ/realtime/sse"
	"github.com/traPtitech/traQ/realtime/viewer"
	"github.com/traPtitech/traQ/realtime/ws"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/message"
	"github.com/traPtitech/traQ/utils/set"
	"go.uber.org/zap"
)

type eventHandler func(ns *Service, ev hub.Message)

var handlerMap = map[string]eventHandler{
	event.MessageCreated:           messageCreatedHandler,
	event.MessageUpdated:           messageUpdatedHandler,
	event.MessageDeleted:           messageDeletedHandler,
	event.MessagePinned:            messagePinnedHandler,
	event.MessageUnpinned:          messageUnpinnedHandler,
	event.MessageStamped:           messageStampedHandler,
	event.MessageUnstamped:         messageUnstampedHandler,
	event.ChannelCreated:           channelCreatedHandler,
	event.ChannelUpdated:           channelUpdatedHandler,
	event.ChannelDeleted:           channelDeletedHandler,
	event.ChannelStared:            channelStaredHandler,
	event.ChannelUnstared:          channelUnstaredHandler,
	event.ChannelRead:              channelReadHandler,
	event.ChannelViewersChanged:    channelViewersChangedHandler,
	event.UserCreated:              userCreatedHandler,
	event.UserUpdated:              userUpdatedHandler,
	event.UserIconUpdated:          userIconUpdatedHandler,
	event.UserOnline:               userOnlineHandler,
	event.UserOffline:              userOfflineHandler,
	event.UserTagAdded:             userTagUpdatedHandler,
	event.UserTagRemoved:           userTagUpdatedHandler,
	event.UserTagUpdated:           userTagUpdatedHandler,
	event.UserGroupCreated:         userGroupCreatedHandler,
	event.UserGroupDeleted:         userGroupDeletedHandler,
	event.UserGroupMemberAdded:     userGroupUpdatedHandler,
	event.UserGroupMemberRemoved:   userGroupUpdatedHandler,
	event.StampCreated:             stampCreatedHandler,
	event.StampUpdated:             stampUpdatedHandler,
	event.StampDeleted:             stampDeletedHandler,
	event.ClipFolderCreated:        clipFolderCreatedHandler,
	event.ClipFolderUpdated:        clipFolderUpdatedHandler,
	event.ClipFolderDeleted:        clipFolderDeletedHandler,
	event.ClipFolderMessageDeleted: clipFolderMessageDeletedHandler,
	event.ClipFolderMessageAdded:   clipFolderMessageAddedHandler,
	event.UserWebRTCStateChanged:   userWebRTCStateChangedHandler,
}

func messageCreatedHandler(ns *Service, ev hub.Message) {
	m := ev.Fields["message"].(*model.Message)
	plain := ev.Fields["plain"].(string)
	embedded := ev.Fields["embedded"].([]*message.EmbeddedInfo)
	logger := ns.logger.With(zap.Stringer("messageId", m.ID))

	// チャンネル情報を取得
	ch, err := ns.repo.GetChannel(m.ChannelID)
	if err != nil {
		logger.Error("failed to GetChannel", zap.Error(err), zap.Stringer("channelId", m.ChannelID)) // 失敗
		return
	}

	// 投稿ユーザー情報を取得
	mUser, err := ns.repo.GetUser(m.UserID, false)
	if err != nil {
		logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("userId", m.UserID)) // 失敗
		return
	}

	fcmPayload := &fcm.Payload{
		Icon: fmt.Sprintf("%s/api/v3/public/icon/%s", ns.origin, strings.ReplaceAll(mUser.GetName(), "#", "%23")),
		Tag:  "c:" + m.ChannelID.String(),
	}
	ssePayload := &sse.EventData{
		EventType: "MESSAGE_CREATED",
		Payload: map[string]interface{}{
			"id": m.ID,
		},
	}

	viewers := set.UUIDSet{}       // バックグラウンドを含む対象チャンネル閲覧中のユーザー
	notifiedUsers := set.UUIDSet{} // チャンネル通知購読ユーザー
	markedUsers := set.UUIDSet{}   // チャンネル未読管理ユーザー
	noticeable := set.UUIDSet{}    // noticeableな未読追加対象のユーザー

	// メッセージボディ作成
	if ch.IsDMChannel() {
		fcmPayload.Title = "@" + mUser.GetResponseDisplayName()
		fcmPayload.Path = "/users/" + mUser.GetName()
		fcmPayload.SetBodyWithEllipsis(plain)
	} else {
		path, err := ns.repo.GetChannelPath(m.ChannelID)
		if err != nil {
			logger.Error("failed to GetChannelPath", zap.Error(err), zap.Stringer("channelId", m.ChannelID))
			return
		}
		fcmPayload.Title = "#" + path
		fcmPayload.Path = "/channels/" + path
		fcmPayload.SetBodyWithEllipsis(mUser.GetResponseDisplayName() + ": " + plain)
	}

	for _, v := range embedded {
		if v.Type == "file" {
			if f, _ := ns.repo.GetFileMeta(uuid.FromStringOrNil(v.ID)); f != nil && f.HasThumbnail() {
				fcmPayload.Image = fmt.Sprintf("%s/api/v3/files/%s/thumbnail", ns.origin, v.ID)
				break
			}
		}
	}

	// 対象者計算
	q := repository.UsersQuery{}.Active().NotBot()
	switch {
	case ch.IsForced: // 強制通知チャンネル
		users, err := ns.repo.GetUserIDs(q)
		if err != nil {
			logger.Error("failed to GetUsers", zap.Error(err)) // 失敗
			return
		}
		notifiedUsers.Add(users...)
		markedUsers.Add(users...)
		noticeable.Add(users...)

	case !ch.IsPublic: // プライベートチャンネル
		users, err := ns.repo.GetUserIDs(q.CMemberOf(ch.ID))
		if err != nil {
			logger.Error("failed to GetPrivateChannelMemberIDs", zap.Error(err), zap.Stringer("channelId", m.ChannelID)) // 失敗
			return
		}
		notifiedUsers.Add(users...)
		markedUsers.Add(users...)

	default: // 通常チャンネルメッセージ
		// チャンネル通知購読者取得
		notify, err := ns.repo.GetUserIDs(q.SubscriberAtNotifyLevelOf(ch.ID))
		if err != nil {
			logger.Error("failed to GetUserIDs", zap.Error(err), zap.Stringer("channelId", m.ChannelID)) // 失敗
			return
		}
		notifiedUsers.Add(notify...)

		// チャンネル未読管理購読者取得
		mark, err := ns.repo.GetUserIDs(q.SubscriberAtMarkLevelOf(ch.ID))
		if err != nil {
			logger.Error("failed to GetUserIDs", zap.Error(err), zap.Stringer("channelId", m.ChannelID)) // 失敗
			return
		}
		markedUsers.Add(mark...)

		// ユーザーグループ・メンションユーザー取得
		for _, v := range embedded {
			switch v.Type {
			case "user":
				if uid, err := uuid.FromString(v.ID); err == nil {
					// TODO 凍結ユーザーの除外
					// MEMO 凍結ユーザーはクライアント側で置換されないのでこのままでも問題はない
					notifiedUsers.Add(uid)
					markedUsers.Add(uid)
					noticeable.Add(uid)
				}
			case "group":
				gs, err := ns.repo.GetUserIDs(q.GMemberOf(uuid.FromStringOrNil(v.ID)))
				if err != nil {
					logger.Error("failed to GetUserGroupMemberIDs", zap.Error(err), zap.String("groupId", v.ID)) // 失敗
					return
				}
				notifiedUsers.Add(gs...)
				markedUsers.Add(gs...)
				noticeable.Add(gs...)
			}
		}
	}

	// チャンネル閲覧者取得
	for uid, swt := range ns.realtime.ViewerManager.GetChannelViewers(m.ChannelID) {
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

	// SSE送信
	for id := range set.UnionUUIDSets(notifiedUsers, viewers) {
		go ns.sse.Multicast(id, ssePayload)
	}

	// WS送信
	go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, ws.TargetUserSets(notifiedUsers, viewers))

	// FCM送信
	if ns.fcm != nil {
		notifiedUsers.Remove(m.UserID)
		ns.fcm.Send(notifiedUsers, fcmPayload)
	}
}

func messageUpdatedHandler(ns *Service, ev hub.Message) {
	channelViewerMulticast(ns, ev.Fields["message"].(*model.Message).ChannelID, &sse.EventData{
		EventType: "MESSAGE_UPDATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["message_id"].(uuid.UUID),
		},
	})
}

func messageDeletedHandler(ns *Service, ev hub.Message) {
	channelViewerMulticast(ns, ev.Fields["message"].(*model.Message).ChannelID, &sse.EventData{
		EventType: "MESSAGE_DELETED",
		Payload: map[string]interface{}{
			"id": ev.Fields["message_id"].(uuid.UUID),
		},
	})
}

func messagePinnedHandler(ns *Service, ev hub.Message) {
	channelViewerMulticast(ns, ev.Fields["channel_id"].(uuid.UUID), &sse.EventData{
		EventType: "MESSAGE_PINNED",
		Payload: map[string]interface{}{
			"id":         ev.Fields["pin_id"].(uuid.UUID),
			"message_id": ev.Fields["message_id"].(uuid.UUID),
			"channel_id": ev.Fields["channel_id"].(uuid.UUID),
		},
	})
}

func messageUnpinnedHandler(ns *Service, ev hub.Message) {
	channelViewerMulticast(ns, ev.Fields["channel_id"].(uuid.UUID), &sse.EventData{
		EventType: "MESSAGE_UNPINNED",
		Payload: map[string]interface{}{
			"id":         ev.Fields["pin_id"].(uuid.UUID),
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

func userWebRTCStateChangedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_WEBRTC_STATE_CHANGED",
		Payload: map[string]interface{}{
			"user_id":    ev.Fields["user_id"].(uuid.UUID),
			"channel_id": ev.Fields["channel_id"].(uuid.UUID),
			"state":      ev.Fields["state"].(set.StringSet),
		},
	})
}

func clipFolderCreatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "CLIP_FOLDER_CREATED",
		Payload: map[string]interface{}{
			"clip_folder_id": ev.Fields["clip_folder_id"].(uuid.UUID),
		},
	})
}

func clipFolderUpdatedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "CLIP_FOLDER_UPDATED",
		Payload: map[string]interface{}{
			"clip_folder_id": ev.Fields["clip_folder_id"].(uuid.UUID),
		},
	})
}

func clipFolderDeletedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "CLIP_FOLDER_DELETED",
		Payload: map[string]interface{}{
			"clip_folder_id": ev.Fields["clip_folder_id"].(uuid.UUID),
		},
	})
}

func clipFolderMessageDeletedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "CLIP_FOLDER_MESSAGE_DELETED",
		Payload: map[string]interface{}{
			"clip_folder_id":         ev.Fields["clip_folder_id"].(uuid.UUID),
			"clip_folder_message_id": ev.Fields["clip_folder_message_id"].(uuid.UUID),
		},
	})
}

func clipFolderMessageAddedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "CLIP_FOLDER_MESSAGE_ADDED",
		Payload: map[string]interface{}{
			"clip_folder_id":         ev.Fields["clip_folder_id"].(uuid.UUID),
			"clip_folder_message_id": ev.Fields["clip_folder_message_id"].(uuid.UUID),
		},
	})
}

func channelHandler(ns *Service, ev hub.Message, ssePayload *sse.EventData) {
	private := ev.Fields["private"].(bool)
	if private {
		cid := ev.Fields["channel_id"].(uuid.UUID)
		members, err := ns.repo.GetPrivateChannelMemberIDs(cid)
		if err != nil {
			ns.logger.Error("failed to GetPrivateChannelMemberIDs", zap.Error(err), zap.Stringer("channelId", cid))
			return
		}
		for _, uid := range members {
			go ns.sse.Multicast(uid, ssePayload)
		}
		go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, ws.TargetUsers(members...))
	} else {
		go ns.sse.Broadcast(ssePayload)
		go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, ws.TargetAll())
	}
}

func channelViewerMulticast(ns *Service, cid uuid.UUID, ssePayload *sse.EventData) {
	for uid := range ns.realtime.ViewerManager.GetChannelViewers(cid) {
		go ns.sse.Multicast(uid, ssePayload)
	}
	go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, ws.TargetChannelViewers(cid))
}

func messageViewerMulticast(ns *Service, mid uuid.UUID, ssePayload *sse.EventData) {
	ch, err := ns.repo.GetChannelByMessageID(mid)
	if err != nil {
		ns.logger.Error("failed to GetChannelByMessageID", zap.Error(err), zap.Stringer("messageId", mid)) // 失敗
		return
	}
	channelViewerMulticast(ns, ch.ID, ssePayload)
}

func broadcast(ns *Service, ssePayload *sse.EventData) {
	go ns.sse.Broadcast(ssePayload)
	go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, ws.TargetAll())
}

func userMulticast(ns *Service, userID uuid.UUID, ssePayload *sse.EventData) {
	go ns.sse.Multicast(userID, ssePayload)
	go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, ws.TargetUsers(userID))
}
