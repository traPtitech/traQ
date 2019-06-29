package bot

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
)

type basePayload struct {
	EventTime time.Time `json:"eventTime"`
}

func makeBasePayload() basePayload {
	return basePayload{
		EventTime: time.Now(),
	}
}

type messagePayload struct {
	ID        uuid.UUID               `json:"id"`
	User      userPayload             `json:"user"`
	ChannelID uuid.UUID               `json:"channelId"`
	Text      string                  `json:"text"`
	PlainText string                  `json:"plainText"`
	Embedded  []*message.EmbeddedInfo `json:"embedded"`
	CreatedAt time.Time               `json:"createdAt"`
	UpdatedAt time.Time               `json:"updatedAt"`
}

func makeMessagePayload(message *model.Message, user *model.User, embedded []*message.EmbeddedInfo, plain string) messagePayload {
	return messagePayload{
		ID:        message.ID,
		User:      makeUserPayload(user),
		ChannelID: message.ChannelID,
		Text:      message.Text,
		PlainText: plain,
		Embedded:  embedded,
		CreatedAt: message.CreatedAt,
		UpdatedAt: message.UpdatedAt,
	}
}

type channelPayload struct {
	ID        uuid.UUID   `json:"id"`
	Name      string      `json:"name"`
	Path      string      `json:"path"`
	ParentID  uuid.UUID   `json:"parentId"`
	Creator   userPayload `json:"creator"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

func makeChannelPayload(ch *model.Channel, path string, user *model.User) channelPayload {
	return channelPayload{
		ID:        ch.ID,
		Name:      ch.Name,
		Path:      "#" + path,
		ParentID:  ch.ParentID,
		Creator:   makeUserPayload(user),
		CreatedAt: ch.CreatedAt,
		UpdatedAt: ch.UpdatedAt,
	}
}

type userPayload struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	IconID      uuid.UUID `json:"iconId"`
	Bot         bool      `json:"bot"`
}

func makeUserPayload(user *model.User) userPayload {
	if user == nil {
		return userPayload{}
	}
	return userPayload{
		ID:          user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		IconID:      user.Icon,
		Bot:         user.Bot,
	}
}

type messageCreatedPayload struct {
	basePayload
	Message messagePayload `json:"message"`
}

type directMessageCreatedPayload struct {
	basePayload
	Message messagePayload `json:"message"`
}

type pingPayload struct {
	basePayload
}

type joinAndLeftPayload struct {
	basePayload
	Channel channelPayload `json:"channel"`
}

type channelCreatedPayload struct {
	basePayload
	Channel channelPayload `json:"channel"`
}

type channelTopicChangedPayload struct {
	basePayload
	Channel channelPayload `json:"channel"`
	Topic   string         `json:"topic"`
	Updater userPayload    `json:"updater"`
}

type userCreatedPayload struct {
	basePayload
	User userPayload `json:"user"`
}

type stampCreatedPayload struct {
	basePayload
	ID      uuid.UUID   `json:"id"`
	Name    string      `json:"name"`
	FileID  uuid.UUID   `json:"fileId"`
	Creator userPayload `json:"creator"`
}
