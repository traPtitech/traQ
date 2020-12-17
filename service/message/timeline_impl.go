package message

import (
	"github.com/gofrs/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
	"time"
)

type timeline struct {
	query       TimelineQuery
	records     []*model.Message
	more        bool
	preloaded   bool
	retrievedAt time.Time
	man         Manager
}

func (t *timeline) Query() TimelineQuery {
	return t.query
}

func (t *timeline) Records() []Message {
	arr := make([]Message, len(t.records))
	for i, record := range t.records {
		arr[i] = &timelineMessage{Model: record, preloaded: t.preloaded}
	}
	return arr
}

func (t *timeline) HasMore() bool {
	return t.more
}

func (t *timeline) RetrievedAt() time.Time {
	return t.retrievedAt
}

type timelineMessage struct {
	Model     *model.Message
	preloaded bool
}

func (m *timelineMessage) GetID() uuid.UUID {
	return m.Model.ID
}

func (m *timelineMessage) GetUserID() uuid.UUID {
	return m.Model.UserID
}

func (m *timelineMessage) GetChannelID() uuid.UUID {
	return m.Model.ChannelID
}

func (m *timelineMessage) GetText() string {
	return m.Model.Text
}

func (m *timelineMessage) GetCreatedAt() time.Time {
	return m.Model.CreatedAt
}

func (m *timelineMessage) GetUpdatedAt() time.Time {
	return m.Model.UpdatedAt
}

func (m *timelineMessage) GetStamps() []model.MessageStamp {
	return m.Model.Stamps
}

func (m *timelineMessage) GetPin() *model.Pin {
	return m.Model.Pin
}

func (m *timelineMessage) MarshalJSON() ([]byte, error) {
	type object struct {
		ID        uuid.UUID `json:"id"`
		UserID    uuid.UUID `json:"userId"`
		ChannelID uuid.UUID `json:"channelId"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"createdAt"`
		UpdatedAt time.Time `json:"updatedAt"`
	}
	type objectWithPreload struct {
		object
		Pinned   bool                 `json:"pinned"`
		Stamps   []model.MessageStamp `json:"stamps"`
		ThreadID optional.UUID        `json:"threadId"` // TODO
	}
	var v interface{}
	if m.preloaded {
		v = &objectWithPreload{
			object: object{
				ID:        m.Model.ID,
				UserID:    m.Model.UserID,
				ChannelID: m.Model.ChannelID,
				Content:   m.Model.Text,
				CreatedAt: m.Model.CreatedAt,
				UpdatedAt: m.Model.UpdatedAt,
			},
			Pinned:   m.Model.Pin != nil,
			Stamps:   m.Model.Stamps,
			ThreadID: optional.UUID{},
		}
	} else {
		v = &object{
			ID:        m.Model.ID,
			UserID:    m.Model.UserID,
			ChannelID: m.Model.ChannelID,
			Content:   m.Model.Text,
			CreatedAt: m.Model.CreatedAt,
			UpdatedAt: m.Model.UpdatedAt,
		}
	}
	return jsoniter.ConfigFastest.Marshal(v)
}
