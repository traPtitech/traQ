package events

type EventType string

const (
	USER_JOINED       EventType = "USER_JOINED"
	USER_LEFT         EventType = "USER_LEFT"
	USER_TAGS_UPDATED EventType = "USER_TAGS_UPDATED"

	CHANNEL_CREATED            EventType = "CHANNEL_CREATED"
	CHANNEL_DELETED            EventType = "CHANNEL_DELETED"
	CHANNEL_RENAMED            EventType = "CHANNEL_RENAMED"
	CHANNEL_STARED             EventType = "CHANNEL_STARED"
	CHANNEL_UNSTARED           EventType = "CHANNEL_UNSTARED"
	CHANNEL_VISIBILITY_CHANGED EventType = "CHANNEL_VISIBILITY_CHANGED"

	MESSAGE_CREATED   EventType = "MESSAGE_CREATED"
	MESSAGE_UPDATED   EventType = "MESSAGE_UPDATED"
	MESSAGE_DELETED   EventType = "MESSAGE_DELETED"
	MESSAGE_READ      EventType = "MESSAGE_READ"
	MESSAGE_STAMPED   EventType = "MESSAGE_STAMPED"
	MESSAGE_UNSTAMPED EventType = "MESSAGE_UNSTAMPED"
	MESSAGE_PINNED    EventType = "MESSAGE_PINNED"
	MESSAGE_UNPINNED  EventType = "MESSAGE_UNPINNED"
	MESSAGE_CLIPPED   EventType = "MESSAGE_CLIPPED"
	MESSAGE_UNCLIPPED EventType = "MESSAGE_UNCLIPPED"

	STAMP_CREATED EventType = "STAMP_CREATED"
	STAMP_DELETED EventType = "STAMP_DELETED"

	TRAQ_UPDATED EventType = "TRAQ_UPDATED"
)

type EventData struct {
	EventType EventType
	Payload   interface{}
}

type UserEvent struct {
	Id string
}

type ChannelEvent struct {
	Id string
}

type UserStarEvent struct {
	ChannelId string
	UserId    string
}

type UserMessageEvent struct {
	UserId    string
	MessageId string
}

type MessageChannelEvent struct {
	MessageId string
	ChannelId string
}

type MessageEvent struct {
	Id        string
	ChannelId string
}

type MessageStampEvent struct {
	Id        string
	ChannelId string
	UserId    string
	StampId   string
	Count     int
}
