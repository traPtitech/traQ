package bot

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
	"time"
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
	UserID    uuid.UUID               `json:"userId"`
	ChannelID uuid.UUID               `json:"channelId"`
	Text      string                  `json:"text"`
	PlainText string                  `json:"plainText"`
	Embedded  []*message.EmbeddedInfo `json:"embedded"`
	CreatedAt time.Time               `json:"createdAt"`
	UpdatedAt time.Time               `json:"updatedAt"`
}

func makeMessagePayload(message *model.Message, embedded []*message.EmbeddedInfo, plain string) messagePayload {
	return messagePayload{
		ID:        message.ID,
		UserID:    message.UserID,
		ChannelID: message.ChannelID,
		Text:      message.Text,
		PlainText: plain,
		Embedded:  embedded,
		CreatedAt: message.CreatedAt,
		UpdatedAt: message.UpdatedAt,
	}
}

type channelPayload struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	ParentID  uuid.UUID `json:"parentId"`
	CreatorID uuid.UUID `json:"creatorId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type userPayload struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	IconID      uuid.UUID `json:"iconId"`
	Bot         bool      `json:"bot"`
}

func makeUserPayload(user *model.User) userPayload {
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

type pingPayload struct {
	basePayload
}

type joinAndLeftPayload struct {
	basePayload
	ChannelID uuid.UUID `json:"channelId"`
}

type channelCreatedPayload struct {
	basePayload
	Channel channelPayload `json:"channel"`
}

type userCreatedPayload struct {
	basePayload
	User userPayload `json:"user"`
}
