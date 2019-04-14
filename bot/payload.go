package bot

import (
	"github.com/gofrs/uuid"
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

type channelPayload struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	ParentID  uuid.UUID `json:"parentId"`
	CreatorID uuid.UUID `json:"creatorId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
	ChannelId uuid.UUID `json:"channelId"`
}

type channelCreatedPayload struct {
	basePayload
	Channel channelPayload `json:"channel"`
}
