package message

import (
	"time"

	"github.com/gofrs/uuid"
	jsonIter "github.com/json-iterator/go"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
)

type timeline struct {
	query       TimelineQuery
	records     []*model.DetailedMessage
	more        bool
	preloaded   bool
	retrievedAt time.Time
	man         Manager
}

// FileInfoOldThumbnail deprecated
type FileInfoOldThumbnail struct {
	Mime   string `json:"mime"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type FileInfoThumbnail struct {
	Type   string `json:"type"`
	Mime   string `json:"mime"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type FileInfo struct {
	ID              uuid.UUID              `json:"id"`
	Name            string                 `json:"name"`
	Mime            string                 `json:"mime"`
	Size            int64                  `json:"size"`
	MD5             string                 `json:"md5"`
	IsAnimatedImage bool                   `json:"isAnimatedImage"`
	CreatedAt       time.Time              `json:"createdAt"`
	Thumbnail       *FileInfoOldThumbnail  `json:"thumbnail"` // deprecated
	ChannelID       optional.Of[uuid.UUID] `json:"channelId"`
	UploaderID      optional.Of[uuid.UUID] `json:"uploaderId"`
	Thumbnails      []FileInfoThumbnail    `json:"thumbnails"`
}

func (t *timeline) Query() TimelineQuery {
	return t.query
}

func (t *timeline) Records() []DetailedMessage {
	arr := make([]DetailedMessage, len(t.records))
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
	Model     *model.DetailedMessage
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

func (m *timelineMessage) GetAttachments() []*model.FileMeta {
	return m.Model.Attachments
}

func (m *timelineMessage) GetQuotes() []*model.QuotedMessage {
	return m.Model.Quotes
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
		Pinned      bool                   `json:"pinned"`
		Stamps      []model.MessageStamp   `json:"stamps"`
		ThreadID    optional.Of[uuid.UUID] `json:"threadId"` // TODO
		Attachments []*FileInfo            `json:"attachments"`
		Quotes      []*quotedMessage       `json:"quotes"`
	}
	var v interface{}
	if m.preloaded {
		quotes := make([]*quotedMessage, len(m.Model.Quotes))
		for i, q := range m.Model.Quotes {
			quotes[i] = &quotedMessage{Model: q}
		}
		tmp := m.Model.Attachments
		fairuinfo := make([]*FileInfo, len(tmp))
		for i, tempu := range tmp {
			samuneiru := make([]FileInfoThumbnail, len(tempu.Thumbnails))
			for j, tn := range tempu.Thumbnails {
				samuneiru[j] = FileInfoThumbnail{
					Type:   tn.Type.String(),
					Mime:   tn.Mime,
					Width:  tn.Width,
					Height: tn.Height,
				}
			}
			fairuinfo[i] = &FileInfo{
				ID:              tempu.ID,
				Name:            tempu.Name,
				Mime:            tempu.Mime,
				Size:            tempu.Size,
				MD5:             tempu.Hash,
				IsAnimatedImage: tempu.IsAnimatedImage,
				CreatedAt:       tempu.CreatedAt,
				ChannelID:       tempu.ChannelID,
				UploaderID:      tempu.CreatorID,
				Thumbnails:      samuneiru,
			}
		}
		v = &objectWithPreload{
			object: object{
				ID:        m.Model.ID,
				UserID:    m.Model.UserID,
				ChannelID: m.Model.ChannelID,
				Content:   m.Model.Text,
				CreatedAt: m.Model.CreatedAt,
				UpdatedAt: m.Model.UpdatedAt,
			},
			Pinned:      m.Model.Pin != nil,
			Stamps:      m.Model.Stamps,
			Attachments: fairuinfo,
			Quotes:      quotes,
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
	return jsonIter.ConfigFastest.Marshal(v)
}
