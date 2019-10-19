package notification

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/fcm"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/realtime"
	"github.com/traPtitech/traQ/realtime/ws"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/sse"
	"github.com/traPtitech/traQ/utils/message"
	"github.com/traPtitech/traQ/utils/set"
	"go.uber.org/zap"
	"strings"
	"time"
)

type eventHandler func(ns *Service, ev hub.Message)

var handlerMap = map[string]eventHandler{
	event.MessageCreated:         messageCreatedHandler,
	event.MessageUpdated:         messageUpdatedHandler,
	event.MessageDeleted:         messageDeletedHandler,
	event.MessagePinned:          messagePinnedHandler,
	event.MessageUnpinned:        messageUnpinnedHandler,
	event.MessageStamped:         messageStampedHandler,
	event.MessageUnstamped:       messageUnstampedHandler,
	event.ChannelCreated:         channelCreatedHandler,
	event.ChannelUpdated:         channelUpdatedHandler,
	event.ChannelDeleted:         channelDeletedHandler,
	event.ChannelStared:          channelStaredHandler,
	event.ChannelUnstared:        channelUnstaredHandler,
	event.ChannelMuted:           channelMutedHandler,
	event.ChannelUnmuted:         channelUnmutedHandler,
	event.ChannelRead:            channelReadHandler,
	event.ChannelViewersChanged:  channelViewersChangedHandler,
	event.UserCreated:            userCreatedHandler,
	event.UserUpdated:            userUpdatedHandler,
	event.UserIconUpdated:        userIconUpdatedHandler,
	event.UserOnline:             userOnlineHandler,
	event.UserOffline:            userOfflineHandler,
	event.UserTagAdded:           userTagAddedHandler,
	event.UserTagRemoved:         userTagRemovedHandler,
	event.UserTagUpdated:         userTagUpdatedHandler,
	event.UserGroupCreated:       userGroupCreatedHandler,
	event.UserGroupDeleted:       userGroupDeletedHandler,
	event.UserGroupMemberAdded:   userGroupMemberAddedHandler,
	event.UserGroupMemberRemoved: userGroupMemberRemovedHandler,
	event.StampCreated:           stampCreatedHandler,
	event.StampUpdated:           stampUpdatedHandler,
	event.StampDeleted:           stampDeletedHandler,
	event.FavoriteStampAdded:     favoriteStampAddedHandler,
	event.FavoriteStampRemoved:   favoriteStampRemovedHandler,
	event.ClipCreated:            clipCreatedHandler,
	event.ClipDeleted:            clipDeletedHandler,
	event.ClipMoved:              clipMovedHandler,
	event.ClipFolderCreated:      clipFolderCreatedHandler,
	event.ClipFolderUpdated:      clipFolderUpdatedHandler,
	event.ClipFolderDeleted:      clipFolderDeletedHandler,
	event.UserWebRTCStateChanged: userWebRTCStateChangedHandler,
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
	mUser, err := ns.repo.GetUser(m.UserID)
	if err != nil {
		logger.Error("failed to GetUser", zap.Error(err), zap.Stringer("userId", m.UserID)) // 失敗
		return
	}
	if len(mUser.DisplayName) == 0 {
		mUser.DisplayName = mUser.Name
	}

	fcmPayload := &fcm.Payload{
		Icon: fmt.Sprintf("%s/api/1.0/public/icon/%s", ns.origin, strings.ReplaceAll(mUser.Name, "#", "%23")),
		Tag:  "c:" + m.ChannelID.String(),
	}
	ssePayload := &sse.EventData{
		EventType: "MESSAGE_CREATED",
		Payload: map[string]interface{}{
			"id": m.ID,
		},
	}
	viewers := set.UUIDSet{}
	connector := set.UUIDSet{}
	subscribers := set.UUIDSet{}
	noticeable := set.UUIDSet{}

	// メッセージボディ作成
	if ch.IsDMChannel() {
		fcmPayload.Title = "@" + mUser.DisplayName
		fcmPayload.Path = "/users/" + mUser.Name
		fcmPayload.SetBodyWithEllipsis(plain)
	} else {
		path, err := ns.repo.GetChannelPath(m.ChannelID)
		if err != nil {
			logger.Error("failed to GetChannelPath", zap.Error(err), zap.Stringer("channelId", m.ChannelID))
			return
		}
		fcmPayload.Title = "#" + path
		fcmPayload.Path = "/channels/" + path
		fcmPayload.SetBodyWithEllipsis(mUser.DisplayName + ": " + plain)
	}

	for _, v := range embedded {
		if v.Type == "file" {
			if f, _ := ns.repo.GetFileMeta(uuid.FromStringOrNil(v.ID)); f != nil && f.HasThumbnail {
				fcmPayload.Image = fmt.Sprintf("%s/api/1.0/files/%s/thumbnail", ns.origin, v.ID)
				break
			}
		}
	}

	// 対象者計算
	targets := set.UUIDSet{}
	q := repository.UsersQuery{}.Active().NotBot()
	switch {
	case ch.IsForced: // 強制通知チャンネル
		users, err := ns.repo.GetUserIDs(q)
		if err != nil {
			logger.Error("failed to GetUsers", zap.Error(err)) // 失敗
			return
		}
		targets.Add(users...)
		subscribers.Add(users...)
		noticeable.Add(users...)

	case !ch.IsPublic: // プライベートチャンネル
		users, err := ns.repo.GetUserIDs(q.CMemberOf(ch.ID))
		if err != nil {
			logger.Error("failed to GetPrivateChannelMemberIDs", zap.Error(err), zap.Stringer("channelId", m.ChannelID)) // 失敗
			return
		}
		targets.Add(users...)
		subscribers.Add(users...)

	default: // 通常チャンネルメッセージ
		users, err := ns.repo.GetUserIDs(q.SubscriberOf(ch.ID))
		if err != nil {
			logger.Error("failed to GetSubscribingUserIDs", zap.Error(err), zap.Stringer("channelId", m.ChannelID)) // 失敗
			return
		}
		targets.Add(users...)
		subscribers.Add(users...)

		// ユーザーグループ・メンションユーザー取得
		for _, v := range embedded {
			switch v.Type {
			case "user":
				if uid, err := uuid.FromString(v.ID); err == nil {
					// TODO 凍結ユーザーの除外
					// MEMO 凍結ユーザーはクライアント側で置換されないのでこのままでも問題はない
					targets.Add(uid)
					subscribers.Add(uid)
					noticeable.Add(uid)
				}
			case "group":
				gs, err := ns.repo.GetUserIDs(q.GMemberOf(uuid.FromStringOrNil(v.ID)))
				if err != nil {
					logger.Error("failed to GetUserGroupMemberIDs", zap.Error(err), zap.String("groupId", v.ID)) // 失敗
					return
				}
				targets.Add(gs...)
				subscribers.Add(gs...)
				noticeable.Add(gs...)
			}
		}

		// ミュート除外
		muted, err := ns.repo.GetMuteUserIDs(m.ChannelID)
		if err != nil {
			logger.Error("failed to GetMuteUserIDs", zap.Error(err), zap.Stringer("channelId", m.ChannelID)) // 失敗
			return
		}
		targets.Remove(muted...)
	}

	// チャンネル閲覧者取得
	for uid, state := range ns.realtime.ViewerManager.GetChannelViewers(m.ChannelID) {
		connector.Add(uid)
		if state > realtime.ViewStateNone {
			viewers.Add(uid)
		}
	}

	targets.Remove(m.UserID) // 自分を除外

	// SSE送信
	for id := range subscribers {
		if !(id == m.UserID || viewers.Contains(id)) {
			err := ns.repo.SetMessageUnread(id, m.ID, noticeable.Contains(id))
			if err != nil {
				logger.Error("failed to SetMessageUnread", zap.Error(err), zap.Stringer("user_id", id)) // 失敗
			}
		}
		go ns.sse.Multicast(id, ssePayload)
	}
	for id := range connector {
		if !subscribers.Contains(id) {
			go ns.sse.Multicast(id, ssePayload)
		}
	}

	// WS送信
	go ns.ws.WriteMessage(ssePayload.EventType, ssePayload.Payload, ws.TargetUserSets(subscribers, connector))

	// FCM送信
	if ns.fcm != nil {
		ns.fcm.Send(targets, fcmPayload)
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
	messageViewerMulticast(ns, ev.Fields["message_id"].(uuid.UUID), &sse.EventData{
		EventType: "MESSAGE_PINNED",
		Payload: map[string]interface{}{
			"id": ev.Fields["pin_id"].(uuid.UUID),
		},
	})
}

func messageUnpinnedHandler(ns *Service, ev hub.Message) {
	messageViewerMulticast(ns, ev.Fields["message_id"].(uuid.UUID), &sse.EventData{
		EventType: "MESSAGE_UNPINNED",
		Payload: map[string]interface{}{
			"id": ev.Fields["pin_id"].(uuid.UUID),
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

func channelMutedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CHANNEL_MUTED",
		Payload: map[string]interface{}{
			"id": ev.Fields["channel_id"].(uuid.UUID),
		},
	})
}

func channelUnmutedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CHANNEL_UNMUTED",
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
	type viewer struct {
		UserID uuid.UUID `json:"userId"`
		State  string    `json:"state"`
	}
	viewers := make([]viewer, 0)
	for uid, state := range ev.Fields["viewers"].(map[uuid.UUID]realtime.ViewState) {
		viewers = append(viewers, viewer{
			UserID: uid,
			State:  state.String(),
		})
	}
	cid := ev.Fields["channel_id"].(uuid.UUID)
	channelViewerMulticast(ns, cid, &sse.EventData{
		EventType: "CHANNEL_VIEWERS_CHANGED",
		Payload: map[string]interface{}{
			"id":      cid,
			"viewers": viewers,
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

func userTagAddedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_TAGS_UPDATED",
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

func userTagRemovedHandler(ns *Service, ev hub.Message) {
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

func userGroupDeletedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_GROUP_DELETED",
		Payload: map[string]interface{}{
			"id": ev.Fields["group_id"].(uuid.UUID),
		},
	})
}

func userGroupMemberAddedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_GROUP_MEMBER_ADDED",
		Payload: map[string]interface{}{
			"id":      ev.Fields["group_id"].(uuid.UUID),
			"user_id": ev.Fields["user_id"].(uuid.UUID),
		},
	})
}

func userGroupMemberRemovedHandler(ns *Service, ev hub.Message) {
	broadcast(ns, &sse.EventData{
		EventType: "USER_GROUP_MEMBER_REMOVED",
		Payload: map[string]interface{}{
			"id":      ev.Fields["group_id"].(uuid.UUID),
			"user_id": ev.Fields["user_id"].(uuid.UUID),
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
		EventType: "STAMP_MODIFIED",
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

func favoriteStampAddedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "FAVORITE_STAMP_ADDED",
		Payload: map[string]interface{}{
			"id": ev.Fields["stamp_id"].(uuid.UUID),
		},
	})
}

func favoriteStampRemovedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "FAVORITE_STAMP_REMOVED",
		Payload: map[string]interface{}{
			"id": ev.Fields["stamp_id"].(uuid.UUID),
		},
	})
}

func clipCreatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CLIP_CREATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["clip_id"].(uuid.UUID),
		},
	})
}

func clipDeletedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CLIP_DELETED",
		Payload: map[string]interface{}{
			"id": ev.Fields["clip_id"].(uuid.UUID),
		},
	})
}

func clipMovedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CLIP_MOVED",
		Payload: map[string]interface{}{
			"id": ev.Fields["clip_id"].(uuid.UUID),
		},
	})
}

func clipFolderCreatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CLIP_FOLDER_CREATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["folder_id"].(uuid.UUID),
		},
	})
}

func clipFolderUpdatedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CLIP_FOLDER_UPDATED",
		Payload: map[string]interface{}{
			"id": ev.Fields["folder_id"].(uuid.UUID),
		},
	})
}

func clipFolderDeletedHandler(ns *Service, ev hub.Message) {
	userMulticast(ns, ev.Fields["user_id"].(uuid.UUID), &sse.EventData{
		EventType: "CLIP_FOLDER_DELETED",
		Payload: map[string]interface{}{
			"id": ev.Fields["folder_id"].(uuid.UUID),
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
