package bot

import "github.com/traPtitech/traQ/model"

const (
	// Ping Pingイベント
	Ping model.BotEvent = "PING"
	// MessageCreated メッセージ作成イベント
	MessageCreated model.BotEvent = "MESSAGE_CREATED"
)

var eventSet = map[model.BotEvent]bool{
	Ping:           true,
	MessageCreated: true,
}

// IsEvent 引数の文字列がボットイベントかどうか
func IsEvent(str string) bool {
	return eventSet[model.BotEvent(str)]
}
