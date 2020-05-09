package event

const (
	// Ping Pingイベント
	Ping Type = "PING"
	// Joined チャンネル参加イベント
	Joined Type = "JOINED"
	// Left チャンネル退出イベント
	Left Type = "LEFT"
	// MessageCreated メッセージ作成イベント
	MessageCreated Type = "MESSAGE_CREATED"
	// MentionMessageCreated メンションメッセージ作成イベント
	MentionMessageCreated Type = "MENTION_MESSAGE_CREATED"
	// DirectMessageCreated ダイレクトメッセージ作成イベント
	DirectMessageCreated Type = "DIRECT_MESSAGE_CREATED"
	// ChannelCreated チャンネル作成イベント
	ChannelCreated Type = "CHANNEL_CREATED"
	// ChannelTopicChanged チャンネルトピック変更イベント
	ChannelTopicChanged Type = "CHANNEL_TOPIC_CHANGED"
	// UserCreated ユーザー作成イベント
	UserCreated Type = "USER_CREATED"
	// StampCreated スタンプ作成イベント
	StampCreated Type = "STAMP_CREATED"
)

var allTypes Types

func init() {
	allTypes = Types{}
	for _, t := range []Type{
		Ping,
		Joined,
		Left,
		MessageCreated,
		MentionMessageCreated,
		DirectMessageCreated,
		ChannelCreated,
		ChannelTopicChanged,
		UserCreated,
		StampCreated,
	} {
		allTypes[t] = struct{}{}
	}
}
