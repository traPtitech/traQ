package events

import (
	"strings"
	"time"

	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/model"
)

//EventType 通知イベントの種類
type EventType string

const (
	//UserJoined ユーザーが新規登録した
	UserJoined EventType = "USER_JOINED"
	//UserLeft ユーザーが脱退した
	UserLeft EventType = "USER_LEFT"
	//UserUpdated ユーザーの情報が更新された
	UserUpdated EventType = "USER_UPDATED"
	//UserTagsUpdated ユーザーのタグが更新された
	UserTagsUpdated EventType = "USER_TAGS_UPDATED"
	//UserIconUpdated ユーザーのアイコンが更新された
	UserIconUpdated EventType = "USER_ICON_UPDATED"

	//ChannelCreated チャンネルが新規作成された
	ChannelCreated EventType = "CHANNEL_CREATED"
	//ChannelDeleted チャンネルが削除された
	ChannelDeleted EventType = "CHANNEL_DELETED"
	//ChannelUpdated チャンネルの名前またはトピックが変更された
	ChannelUpdated EventType = "CHANNEL_UPDATED"
	//ChannelStared チャンネルをスターした
	ChannelStared EventType = "CHANNEL_STARED"
	//ChannelUnstared チャンネルのスターを解除した
	ChannelUnstared EventType = "CHANNEL_UNSTARED"
	//ChannelVisibilityChanged チャンネルの可視状態が変更された
	ChannelVisibilityChanged EventType = "CHANNEL_VISIBILITY_CHANGED"

	//MessageCreated メッセージが投稿された
	MessageCreated EventType = "MESSAGE_CREATED"
	//MessageUpdated メッセージが更新された
	MessageUpdated EventType = "MESSAGE_UPDATED"
	//MessageDeleted メッセージが削除された
	MessageDeleted EventType = "MESSAGE_DELETED"
	//MessageRead メッセージを読んだ
	MessageRead EventType = "MESSAGE_READ"
	//MessageStamped メッセージにスタンプが押された
	MessageStamped EventType = "MESSAGE_STAMPED"
	//MessageUnstamped メッセージからスタンプが外された
	MessageUnstamped EventType = "MESSAGE_UNSTAMPED"
	//MessagePinned メッセージがピン留めされた
	MessagePinned EventType = "MESSAGE_PINNED"
	//MessageUnpinned ピン留めされたメッセージのピンが外された
	MessageUnpinned EventType = "MESSAGE_UNPINNED"

	//ClipCreated メッセージをクリップした
	ClipCreated EventType = "CLIP_CREATED"
	//ClipDeleted メッセージをアンクリップした
	ClipDeleted EventType = "CLIP_DELETED"
	//ClipMoved クリップのフォルダが変更された
	ClipMoved EventType = "CLIP_MOVED"
	//ClipFolderCreated クリップフォルダが作成された
	ClipFolderCreated EventType = "CLIP_FOLDER_CREATED"
	//ClipFolderUpdated クリップフォルダが更新された
	ClipFolderUpdated EventType = "CLIP_FOLDER_UPDATED"
	//ClipFolderDeleted クリップフォルダが削除された
	ClipFolderDeleted EventType = "CLIP_FOLDER_DELETED"

	//StampCreated スタンプが新しく追加された
	StampCreated EventType = "STAMP_CREATED"
	//StampModified スタンプが修正された
	StampModified EventType = "STAMP_MODIFIED"
	//StampDeleted スタンプが削除された
	StampDeleted EventType = "STAMP_DELETED"

	//TraqUpdated traQが更新された
	TraqUpdated EventType = "TRAQ_UPDATED"
)

// DataPayload データペイロード型
type DataPayload map[string]interface{}

// Event イベントのインターフェイス
type Event interface {
	DataPayload() DataPayload
}

// UserTargetEvent 特定のユーザー宛のイベントのインターフェイス
type UserTargetEvent interface {
	Event
	TargetUser() uuid.UUID
}

// ChannelUserTargetEvent 特定のチャンネルを見ているユーザー宛のイベントのインターフェイス
type ChannelUserTargetEvent interface {
	Event
	TargetChannel() uuid.UUID
}

// UserEvent ユーザーに関するイベント
type UserEvent struct {
	ID string
}

// DataPayload データペイロード
func (e UserEvent) DataPayload() DataPayload {
	return DataPayload{
		"id": e.ID,
	}
}

// ChannelEvent チャンネルに関するイベント
type ChannelEvent struct {
	ID string
}

// DataPayload データペイロード
func (e ChannelEvent) DataPayload() DataPayload {
	return DataPayload{
		"id": e.ID,
	}
}

// UserChannelEvent ユーザーとチャンネルに関するイベント
type UserChannelEvent struct {
	UserID    string
	ChannelID string
}

// DataPayload データペイロード
func (e UserChannelEvent) DataPayload() DataPayload {
	return DataPayload{
		"id": e.ChannelID,
	}
}

// TargetUser 通知対象のユーザーID
func (e UserChannelEvent) TargetUser() uuid.UUID {
	return uuid.FromStringOrNil(e.UserID)
}

// UserMessageEvent ユーザーとメッセージに関するイベント
type UserMessageEvent struct {
	UserID    string
	MessageID string
}

// DataPayload データペイロード
func (e UserMessageEvent) DataPayload() DataPayload {
	return DataPayload{
		"id": e.MessageID,
	}
}

// TargetUser 通知対象のユーザーID
func (e UserMessageEvent) TargetUser() uuid.UUID {
	return uuid.FromStringOrNil(e.UserID)
}

// ReadMessagesEvent メッセージの既読イベント
type ReadMessagesEvent struct {
	UserID    string
	ChannelID string
}

// DataPayload データペイロード
func (e ReadMessagesEvent) DataPayload() DataPayload {
	return DataPayload{
		"ids": e.ChannelID,
	}
}

// TargetUser 通知対象のユーザーID
func (e ReadMessagesEvent) TargetUser() uuid.UUID {
	return uuid.FromStringOrNil(e.UserID)
}

// PinEvent メッセージのピンイベント
type PinEvent struct {
	PinID   string
	Message model.Message
}

// DataPayload データペイロード
func (e PinEvent) DataPayload() DataPayload {
	return DataPayload{
		"id": e.PinID,
	}
}

// TargetChannel 通知対象のチャンネル
func (e PinEvent) TargetChannel() uuid.UUID {
	return uuid.FromStringOrNil(e.Message.ChannelID)
}

// MessageEvent メッセージに関するイベント
type MessageEvent struct {
	Message model.Message
}

// DataPayload データペイロード
func (e MessageEvent) DataPayload() DataPayload {
	path, _ := model.GetChannelPath(uuid.FromStringOrNil(e.Message.ChannelID))
	return DataPayload{
		"id":           e.Message.ID,
		"channel_path": strings.TrimLeft(path, "#"),
	}
}

// TargetChannel 通知対象のチャンネル
func (e MessageEvent) TargetChannel() uuid.UUID {
	return uuid.FromStringOrNil(e.Message.ChannelID)
}

// MessageStampEvent メッセージとスタンプに関するイベント
type MessageStampEvent struct {
	ID        string
	ChannelID string
	UserID    string
	StampID   string
	Count     int
	CreatedAt time.Time
}

// DataPayload データペイロード
func (e MessageStampEvent) DataPayload() DataPayload {
	if e.Count > 0 {
		return DataPayload{
			"message_id": e.ID,
			"user_id":    e.UserID,
			"stamp_id":   e.StampID,
			"count":      e.Count,
			"created_at": e.CreatedAt,
		}
	}
	return DataPayload{
		"message_id": e.ID,
		"user_id":    e.UserID,
		"stamp_id":   e.StampID,
	}
}

// TargetChannel 通知対象のチャンネル
func (e MessageStampEvent) TargetChannel() uuid.UUID {
	return uuid.FromStringOrNil(e.ChannelID)
}

// StampEvent スタンプに関するイベント
type StampEvent struct {
	ID string
}

// DataPayload データペイロード
func (e StampEvent) DataPayload() DataPayload {
	return DataPayload{
		"id": e.ID,
	}
}

// ClipEvent クリップに関するイベント
type ClipEvent struct {
	ID     string
	UserID string
}

// DataPayload データペイロード
func (e ClipEvent) DataPayload() DataPayload {
	return DataPayload{
		"id": e.ID,
	}
}

// TargetUser 通知対象のユーザーID
func (e ClipEvent) TargetUser() uuid.UUID {
	return uuid.FromStringOrNil(e.UserID)
}
