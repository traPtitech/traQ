package event

import (
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/config"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
	"golang.org/x/exp/utf8string"
	"strings"
	"sync"
	"time"
)

// Payload データペイロード型
type Payload map[string]interface{}

// MessageCreatedEvent メッセージ作成イベントデータ
type MessageCreatedEvent struct {
	Message  model.Message
	plain    string
	entities []*message.EmbeddedInfo
	lock     sync.Mutex
}

func (e *MessageCreatedEvent) parseMessage() ([]*message.EmbeddedInfo, string) {
	e.lock.Lock()
	defer e.lock.Unlock()
	if len(e.plain) > 0 {
		return e.entities, e.plain
	}
	e.entities, e.plain = message.Parse(e.Message.Text)
	return e.entities, e.plain
}

// GetTargetUsers 通知対象のユーザー
func (e *MessageCreatedEvent) GetTargetUsers() map[uuid.UUID]bool {
	res := map[uuid.UUID]bool{}
	ch, err := model.GetChannelByMessageID(e.Message.GetID())
	if err != nil {
		log.Error(err)
	}

	switch {
	case ch.IsForced: // 強制通知チャンネル
		users, err := model.GetUsers()
		if err != nil {
			log.Error(err)
		}
		for _, v := range users {
			if v.Bot {
				continue
			}
			res[v.GetUID()] = true
		}

	case !ch.IsPublic: // プライベートチャンネル
		users, err := model.GetPrivateChannelMembers(ch.GetCID())
		if err != nil {
			log.Error(err)
		}
		for _, v := range users {
			res[v] = true
		}

	default: // 通常チャンネルメッセージ
		// チャンネル通知ユーザー取得
		for k := range e.GetTargetChannels() {
			users, err := model.GetSubscribingUser(k)
			if err != nil {
				log.Error(err)
			}
			for _, v := range users {
				res[v] = true
			}
		}

		// タグユーザー・メンションユーザー取得
		ei, _ := e.parseMessage()
		for _, v := range ei {
			switch v.Type {
			case "user":
				if uid, err := uuid.FromString(v.ID); err != nil {
					res[uid] = true
				}
			case "tag":
				tagged, err := model.GetUserIDsByTagID(uuid.FromStringOrNil(v.ID))
				if err != nil {
					log.Error(err)
				}
				for _, v := range tagged {
					res[v] = true
				}
			}
		}
	}

	return res
}

// GetExcludeUsers FCMの通知対象から除外されるユーザー
func (e *MessageCreatedEvent) GetExcludeUsers() map[uuid.UUID]bool {
	ex := map[uuid.UUID]bool{e.Message.GetUID(): true}

	ch, err := model.GetChannelByMessageID(e.Message.GetID())
	if err != nil {
		log.Error(err)
	}
	if !ch.IsForced {
		muted, err := model.GetMuteUserIDs(ch.GetCID())
		if err != nil {
			log.Error(err)
		}
		for _, v := range muted {
			ex[uuid.Must(uuid.FromString(v))] = true
		}
	}

	return ex
}

// GetFCMData FCM用のペイロード
func (e *MessageCreatedEvent) GetFCMData() map[string]string {
	d := map[string]string{
		"icon":      fmt.Sprintf("%s/api/1.0/users/%s/icon?thumb", config.TRAQOrigin, e.Message.UserID),
		"vibration": "[1000, 1000, 1000]",
		"tag":       fmt.Sprintf("c:%s", e.Message.ChannelID),
		"badge":     fmt.Sprintf("%s/static/badge.png", config.TRAQOrigin),
	}

	ei, plain := e.parseMessage()
	users, _ := model.GetPrivateChannelMembers(e.Message.GetCID())
	mUser, _ := model.GetUser(e.Message.GetUID())
	if l := len(users); l == 2 || l == 1 {
		if mUser != nil {
			if len(mUser.DisplayName) == 0 {
				d["title"] = fmt.Sprintf("@%s", mUser.Name)
			} else {
				d["title"] = fmt.Sprintf("@%s", mUser.DisplayName)
			}
			d["path"] = fmt.Sprintf("/users/%s", mUser.Name)
		} else {
			d["title"] = fmt.Sprintf("traQ")
		}

		if s := utf8string.NewString(plain); s.RuneCount() > 100 {
			d["body"] = s.Slice(0, 97) + "..."
		} else {
			d["body"] = plain
		}
	} else {
		path, _ := model.GetChannelPath(e.Message.GetCID())
		d["title"] = path
		d["path"] = fmt.Sprintf("/channels/%s", strings.TrimLeft(path, "#"))

		body := ""
		if mUser != nil {
			if len(mUser.DisplayName) == 0 {
				body = fmt.Sprintf("%s: %s", mUser.Name, plain)
			} else {
				body = fmt.Sprintf("%s: %s", mUser.DisplayName, plain)
			}
		} else {
			body = fmt.Sprintf("[ユーザー名が取得できませんでした]: %s", plain)
		}
		if s := utf8string.NewString(body); s.RuneCount() > 100 {
			d["body"] = s.Slice(0, 97) + "..."
		} else {
			d["body"] = body
		}
	}

	for _, v := range ei {
		if v.Type == "file" {
			f, _ := model.GetMetaFileDataByID(uuid.FromStringOrNil(v.ID))
			if f != nil && f.HasThumbnail {
				d["image"] = fmt.Sprintf("%s/api/1.0/files/%s/thumbnail", config.TRAQOrigin, v.ID)
				break
			}
		}
	}

	return d
}

// GetTargetChannels 通知対象のチャンネル
func (e *MessageCreatedEvent) GetTargetChannels() map[uuid.UUID]bool {
	return map[uuid.UUID]bool{e.Message.GetCID(): true}
}

// GetData SSE用のペイロード
func (e *MessageCreatedEvent) GetData() Payload {
	return Payload{
		"id": e.Message.ID,
	}
}

// GetBotPayload BOT用のペイロード
func (e *MessageCreatedEvent) GetBotPayload() interface{} {
	return Payload{
		"messageId": e.Message.ID,
		"userId":    e.Message.UserID,
		"channelId": e.Message.ChannelID,
		"content":   e.Message.Text,
		"createdAt": e.Message.CreatedAt,
		"updatedAt": e.Message.UpdatedAt,
	}
}

// MessageUpdatedEvent メッセージ更新イベントデータ
type MessageUpdatedEvent struct {
	Message model.Message
}

// GetTargetChannels 通知対象のチャンネル
func (e *MessageUpdatedEvent) GetTargetChannels() map[uuid.UUID]bool {
	return map[uuid.UUID]bool{e.Message.GetCID(): true}
}

// GetData SSE用のペイロード
func (e *MessageUpdatedEvent) GetData() Payload {
	return Payload{
		"id": e.Message.ID,
	}
}

// GetBotPayload BOT用のペイロード
func (e *MessageUpdatedEvent) GetBotPayload() interface{} {
	return Payload{
		"messageId": e.Message.ID,
		"userId":    e.Message.UserID,
		"channelId": e.Message.ChannelID,
		"content":   e.Message.Text,
		"createdAt": e.Message.CreatedAt,
		"updatedAt": e.Message.UpdatedAt,
	}
}

// MessageDeletedEvent メッセージ削除イベントデータ
type MessageDeletedEvent struct {
	Message model.Message
}

// GetBotPayload BOT用のペイロード
func (e *MessageDeletedEvent) GetBotPayload() interface{} {
	return Payload{
		"messageId": e.Message.ID,
		"channelId": e.Message.ChannelID,
	}
}

// GetData SSE用のペイロード
func (e *MessageDeletedEvent) GetData() Payload {
	return Payload{
		"id": e.Message.ID,
	}
}

// GetTargetChannels 通知対象のチャンネル
func (e *MessageDeletedEvent) GetTargetChannels() map[uuid.UUID]bool {
	return map[uuid.UUID]bool{e.Message.GetCID(): true}
}

// UserEvent ユーザーイベント
type UserEvent struct {
	ID string
}

// GetData SSE用のペイロード
func (e *UserEvent) GetData() Payload {
	return Payload{
		"id": e.ID,
	}
}

// ChannelEvent チャンネルイベント
type ChannelEvent struct {
	ID string
}

// GetData SSE用のペイロード
func (e *ChannelEvent) GetData() Payload {
	return Payload{
		"id": e.ID,
	}
}

// PrivateChannelEvent プライベートチャンネルに関するイベント
type PrivateChannelEvent struct {
	ChannelID uuid.UUID
}

// GetTargetUsers 通知対象のユーザー
func (e *PrivateChannelEvent) GetTargetUsers() map[uuid.UUID]bool {
	members, err := model.GetPrivateChannelMembers(e.ChannelID)
	if err != nil {
		log.Error(err)
	}
	res := map[uuid.UUID]bool{}
	for _, v := range members {
		res[v] = true
	}
	return res
}

// GetData SSE用のペイロード
func (e *PrivateChannelEvent) GetData() Payload {
	return Payload{
		"id": e.ChannelID.String(),
	}
}

// UserChannelEvent ユーザーとチャンネルに関するイベント
type UserChannelEvent struct {
	UserID    uuid.UUID
	ChannelID uuid.UUID
}

// GetTargetUsers 通知対象のユーザー
func (e *UserChannelEvent) GetTargetUsers() map[uuid.UUID]bool {
	return map[uuid.UUID]bool{e.UserID: true}
}

// GetData SSE用のペイロード
func (e *UserChannelEvent) GetData() Payload {
	return Payload{
		"id": e.ChannelID.String(),
	}
}

// StampEvent スタンプに関するイベント
type StampEvent struct {
	ID uuid.UUID
}

// GetData SSE用のペイロード
func (e *StampEvent) GetData() Payload {
	return Payload{
		"id": e.ID.String(),
	}
}

// MessageStampEvent メッセージとスタンプに関するイベント
type MessageStampEvent struct {
	ID        uuid.UUID
	ChannelID uuid.UUID
	UserID    uuid.UUID
	StampID   uuid.UUID
	Count     int
	CreatedAt time.Time
}

// GetData SSE用のペイロード
func (e *MessageStampEvent) GetData() Payload {
	if e.Count > 0 {
		return Payload{
			"message_id": e.ID.String(),
			"user_id":    e.UserID.String(),
			"stamp_id":   e.StampID.String(),
			"count":      e.Count,
			"created_at": e.CreatedAt,
		}
	}
	return Payload{
		"message_id": e.ID.String(),
		"user_id":    e.UserID.String(),
		"stamp_id":   e.StampID.String(),
	}
}

// GetTargetChannels 通知対象のチャンネル
func (e *MessageStampEvent) GetTargetChannels() map[uuid.UUID]bool {
	return map[uuid.UUID]bool{e.ChannelID: true}
}

// ReadMessageEvent メッセージの既読イベント
type ReadMessageEvent struct {
	UserID    uuid.UUID
	ChannelID uuid.UUID
}

// GetTargetUsers 通知対象のユーザー
func (e *ReadMessageEvent) GetTargetUsers() map[uuid.UUID]bool {
	return map[uuid.UUID]bool{e.UserID: true}
}

// GetData SSE用のペイロード
func (e *ReadMessageEvent) GetData() Payload {
	return Payload{
		"id": e.ChannelID.String(),
	}
}

// PinEvent メッセージのピンイベント
type PinEvent struct {
	PinID   uuid.UUID
	Message model.Message
}

// GetData SSE用のペイロード
func (e *PinEvent) GetData() Payload {
	return Payload{
		"id": e.PinID.String(),
	}
}

// GetTargetChannels 通知対象のチャンネル
func (e *PinEvent) GetTargetChannels() map[uuid.UUID]bool {
	return map[uuid.UUID]bool{uuid.Must(uuid.FromString(e.Message.ChannelID)): true}
}

// ClipEvent クリップに関するイベント
type ClipEvent struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

// GetData SSE用のペイロード
func (e *ClipEvent) GetData() Payload {
	return Payload{
		"id": e.ID.String(),
	}
}

// GetTargetUsers 通知対象のユーザー
func (e *ClipEvent) GetTargetUsers() map[uuid.UUID]bool {
	return map[uuid.UUID]bool{e.UserID: true}
}
