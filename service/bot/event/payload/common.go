package payload

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/message"
)

// Base 全イベントに埋め込まれるペイロード
type Base struct {
	EventTime time.Time `json:"eventTime"`
}

func MakeBase(et time.Time) Base {
	return Base{
		EventTime: et,
	}
}

type Message struct {
	ID        uuid.UUID               `json:"id"`
	User      User                    `json:"user"`
	ChannelID uuid.UUID               `json:"channelId"`
	Text      string                  `json:"text"`
	PlainText string                  `json:"plainText"`
	Embedded  []*message.EmbeddedInfo `json:"embedded"`
	CreatedAt time.Time               `json:"createdAt"`
	UpdatedAt time.Time               `json:"updatedAt"`
}

func MakeMessage(message *model.Message, user model.UserInfo, embedded []*message.EmbeddedInfo, plain string) Message {
	return Message{
		ID:        message.ID,
		User:      MakeUser(user),
		ChannelID: message.ChannelID,
		Text:      message.Text,
		PlainText: plain,
		Embedded:  embedded,
		CreatedAt: message.CreatedAt,
		UpdatedAt: message.UpdatedAt,
	}
}

type Channel struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	ParentID  uuid.UUID `json:"parentId"`
	Creator   User      `json:"creator"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func MakeChannel(ch *model.Channel, path string, user model.UserInfo) Channel {
	return Channel{
		ID:        ch.ID,
		Name:      ch.Name,
		Path:      "#" + path,
		ParentID:  ch.ParentID,
		Creator:   MakeUser(user),
		CreatedAt: ch.CreatedAt,
		UpdatedAt: ch.UpdatedAt,
	}
}

type User struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	IconID      uuid.UUID `json:"iconId"`
	Bot         bool      `json:"bot"`
}

func MakeUser(user model.UserInfo) User {
	if user == nil {
		return User{}
	}

	payload := User{
		ID:          user.GetID(),
		Name:        user.GetName(),
		DisplayName: user.GetResponseDisplayName(),
		IconID:      user.GetIconFileID(),
		Bot:         user.IsBot(),
	}
	return payload
}
