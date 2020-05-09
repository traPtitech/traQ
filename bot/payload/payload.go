package payload

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
)

type Ping struct {
	Base
}

func MakePing() *Ping {
	return &Ping{Base: MakeBase()}
}

type MessageCreated struct {
	Base
	Message Message `json:"message"`
}

func MakeMessageCreated(m *model.Message, user model.UserInfo, embedded []*message.EmbeddedInfo, parsed *message.ParseResult) *MessageCreated {
	return &MessageCreated{
		Base:    MakeBase(),
		Message: MakeMessage(m, user, embedded, parsed.PlainText),
	}
}

type DirectMessageCreated struct {
	Base
	Message Message `json:"message"`
}

func MakeDirectMessageCreated(m *model.Message, user model.UserInfo, embedded []*message.EmbeddedInfo, parsed *message.ParseResult) *DirectMessageCreated {
	return &DirectMessageCreated{
		Base:    MakeBase(),
		Message: MakeMessage(m, user, embedded, parsed.PlainText),
	}
}

type JoinedOrLeft struct {
	Base
	Channel Channel `json:"channel"`
}

func MakeJoinedOrLeft(ch *model.Channel, chPath string, user model.UserInfo) *JoinedOrLeft {
	return &JoinedOrLeft{
		Base:    MakeBase(),
		Channel: MakeChannel(ch, chPath, user),
	}
}

type ChannelCreated struct {
	Base
	Channel Channel `json:"channel"`
}

func MakeChannelCreated(ch *model.Channel, chPath string, user model.UserInfo) *ChannelCreated {
	return &ChannelCreated{
		Base:    MakeBase(),
		Channel: MakeChannel(ch, chPath, user),
	}
}

type ChannelTopicChanged struct {
	Base
	Channel Channel `json:"channel"`
	Topic   string  `json:"topic"`
	Updater User    `json:"updater"`
}

func MakeChannelTopicChanged(ch *model.Channel, chPath string, chCreator model.UserInfo, topic string, user model.UserInfo) *ChannelTopicChanged {
	return &ChannelTopicChanged{
		Base:    MakeBase(),
		Channel: MakeChannel(ch, chPath, chCreator),
		Topic:   topic,
		Updater: MakeUser(user),
	}
}

type UserCreated struct {
	Base
	User User `json:"user"`
}

func MakeUserCreated(user model.UserInfo) *UserCreated {
	return &UserCreated{
		Base: MakeBase(),
		User: MakeUser(user),
	}
}

type StampCreated struct {
	Base
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	FileID  uuid.UUID `json:"fileId"`
	Creator User      `json:"creator"`
}

func MakeStampCreated(stamp *model.Stamp, user model.UserInfo) *StampCreated {
	return &StampCreated{
		Base:    MakeBase(),
		ID:      stamp.ID,
		Name:    stamp.Name,
		FileID:  stamp.FileID,
		Creator: MakeUser(user),
	}
}
