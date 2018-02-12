package events

import "github.com/traPtitech/traQ/model"

type EventType string

const (
	UserJoined      EventType = "USER_JOINED"
	UserLeft        EventType = "USER_LEFT"
	UserTagsUpdated EventType = "USER_TAGS_UPDATED"

	ChannelCreated           EventType = "CHANNEL_CREATED"
	ChannelDeleted           EventType = "CHANNEL_DELETED"
	ChannelUpdated           EventType = "CHANNEL_UPDATED"
	ChannelStared            EventType = "CHANNEL_STARED"
	ChannelUnstared          EventType = "CHANNEL_UNSTARED"
	ChannelVisibilityChanged EventType = "CHANNEL_VISIBILITY_CHANGED"

	MessageCreated   EventType = "MESSAGE_CREATED"
	MessageUpdated   EventType = "MESSAGE_UPDATED"
	MessageDeleted   EventType = "MESSAGE_DELETED"
	MessageRead      EventType = "MESSAGE_READ"
	MessageStamped   EventType = "MESSAGE_STAMPED"
	MessageUnstamped EventType = "MESSAGE_UNSTAMPED"
	MessagePinned    EventType = "MESSAGE_PINNED"
	MessageUnpinned  EventType = "MESSAGE_UNPINNED"
	MessageClipped   EventType = "MESSAGE_CLIPPED"
	MessageUnclipped EventType = "MESSAGE_UNCLIPPED"

	StampCreated EventType = "STAMP_CREATED"
	StampDeleted EventType = "STAMP_DELETED"

	TraqUpdated EventType = "TRAQ_UPDATED"
)

type EventData struct {
	EventType EventType
	Summary   string
	Payload   interface{}
	Mobile    bool
}

type UserEvent struct {
	Id string
}

type ChannelEvent struct {
	Id string
}

type UserChannelEvent struct {
	UserId    string
	ChannelId string
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
	Message model.Message
}

type MessageStampEvent struct {
	Id        string
	ChannelId string
	UserId    string
	StampId   string
	Count     int
}
