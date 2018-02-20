package events

import "github.com/traPtitech/traQ/model"

//EventType 通知イベントの種類
type EventType string

const (
	//UserJoined ユーザーが新規登録した
	UserJoined EventType = "USER_JOINED"
	//UserLeft ユーザーが脱退した
	UserLeft EventType = "USER_LEFT"
	//UserTagsUpdated ユーザーのタグが更新された
	UserTagsUpdated EventType = "USER_TAGS_UPDATED"

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
	//MessageClipped メッセージをクリップした
	MessageClipped EventType = "MESSAGE_CLIPPED"
	//MessageUnclipped メッセージをアンクリップした
	MessageUnclipped EventType = "MESSAGE_UNCLIPPED"

	//StampCreated スタンプが新しく追加された
	StampCreated EventType = "STAMP_CREATED"
	//StampDeleted スタンプが削除された
	StampDeleted EventType = "STAMP_DELETED"

	//TraqUpdated traQが更新された
	TraqUpdated EventType = "TRAQ_UPDATED"
)

//UserEvent ユーザーに関するイベントのペイロード
type UserEvent struct {
	ID string
}

//ChannelEvent チャンネルに関するイベントのペイロード
type ChannelEvent struct {
	ID string
}

//UserChannelEvent ユーザーとチャンネルに関するイベントのペイロード
type UserChannelEvent struct {
	UserID    string
	ChannelID string
}

//UserMessageEvent ユーザーとメッセージに関するイベントのペイロード
type UserMessageEvent struct {
	UserID    string
	MessageID string
}

//ReadMessagesEvent メッセージの既読イベントのペイロード
type ReadMessagesEvent struct {
	UserID     string
	MessageIDs []string
}

//MessageChannelEvent メッセージとチャンネルに関するイベントのペイロード
type MessageChannelEvent struct {
	MessageID string
	ChannelID string
}

//MessageEvent メッセージに関するイベントのペイロード
type MessageEvent struct {
	Message model.Message
}

//MessageStampEvent メッセージとスタンプに関するイベントのペイロード
type MessageStampEvent struct {
	ID        string
	ChannelID string
	UserID    string
	StampID   string
	Count     int
}
