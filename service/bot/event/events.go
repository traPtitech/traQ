package event

import "github.com/traPtitech/traQ/model"

const (
	// Ping Pingイベント
	Ping model.BotEventType = "PING"
	// Joined チャンネル参加イベント
	Joined model.BotEventType = "JOINED"
	// Left チャンネル退出イベント
	Left model.BotEventType = "LEFT"
	// MessageCreated メッセージ作成イベント
	MessageCreated model.BotEventType = "MESSAGE_CREATED"
	// MessageDeleted メッセージ削除イベント
	MessageDeleted model.BotEventType = "MESSAGE_DELETED"
	// MessageUpdated メッセージ編集イベント
	MessageUpdated model.BotEventType = "MESSAGE_UPDATED"
	// BotMessageStampsUpdated BOTメッセージスタンプ更新イベント
	BotMessageStampsUpdated model.BotEventType = "BOT_MESSAGE_STAMPS_UPDATED"
	// MentionMessageCreated メンションメッセージ作成イベント
	MentionMessageCreated model.BotEventType = "MENTION_MESSAGE_CREATED"
	// DirectMessageCreated ダイレクトメッセージ作成イベント
	DirectMessageCreated model.BotEventType = "DIRECT_MESSAGE_CREATED"
	// DirectMessageUpdated ダイレクトメッセージ編集イベント
	DirectMessageUpdated model.BotEventType = "DIRECT_MESSAGE_UPDATED"
	// DirectMessageDeleted ダイレクトメッセージ削除イベント
	DirectMessageDeleted model.BotEventType = "DIRECT_MESSAGE_DELETED"
	// ChannelCreated チャンネル作成イベント
	ChannelCreated model.BotEventType = "CHANNEL_CREATED"
	// ChannelTopicChanged チャンネルトピック変更イベント
	ChannelTopicChanged model.BotEventType = "CHANNEL_TOPIC_CHANGED"
	// UserCreated ユーザー作成イベント
	UserCreated model.BotEventType = "USER_CREATED"
	// StampCreated スタンプ作成イベント
	StampCreated model.BotEventType = "STAMP_CREATED"
	// TagAdded タグ追加イベント
	TagAdded model.BotEventType = "TAG_ADDED"
	// TagUpdated タグ編集イベント
	TagUpdated model.BotEventType = "TAG_UPDATED"
	// TagRemoved タグ削除イベント
	TagRemoved model.BotEventType = "TAG_REMOVED"
)

var Types model.BotEventTypes

func init() {
	Types = model.BotEventTypes{}
	for _, t := range []model.BotEventType{
		// ここに全てのBOTイベントを入れてください
		Ping,
		Joined,
		Left,
		MessageCreated,
		MessageDeleted,
		MessageUpdated,
		BotMessageStampsUpdated,
		MentionMessageCreated,
		DirectMessageCreated,
		DirectMessageUpdated,
		DirectMessageDeleted,
		ChannelCreated,
		ChannelTopicChanged,
		UserCreated,
		StampCreated,
		TagAdded,
		TagRemoved,
	} {
		Types[t] = struct{}{}
	}
}
