package bot

import "github.com/traPtitech/traQ/model"

const (
	// Ping Pingイベント
	Ping model.BotEvent = "PING"
	// Joined チャンネル参加イベント
	Joined model.BotEvent = "JOINED"
	// Left チャンネル退出イベント
	Left model.BotEvent = "LEFT"

	// MessageCreated メッセージ作成イベント
	MessageCreated model.BotEvent = "MESSAGE_CREATED"

	// DirectMessageCreated ダイレクトメッセージ作成イベント
	DirectMessageCreated model.BotEvent = "DIRECT_MESSAGE_CREATED"

	// ChannelCreated チャンネル作成イベント
	ChannelCreated model.BotEvent = "CHANNEL_CREATED"

	// UserCreated ユーザー作成イベント
	UserCreated model.BotEvent = "USER_CREATED"
)

var eventSet = map[model.BotEvent]bool{
	Ping:                 true,
	Joined:               true,
	Left:                 true,
	MessageCreated:       true,
	DirectMessageCreated: true,
	ChannelCreated:       true,
	UserCreated:          true,
}

// IsEvent 引数の文字列がボットイベントかどうか
func IsEvent(str string) bool {
	return eventSet[model.BotEvent(str)]
}
