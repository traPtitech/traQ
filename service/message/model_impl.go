package message

import (
	"sync"
	"time"

	"github.com/gofrs/uuid"
	jsonIter "github.com/json-iterator/go"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/optional"
)

type message struct {
	Model *model.Message

	// StampId -> (UserId -> MessageStamp)
	stampMap      map[uuid.UUID]map[uuid.UUID]model.MessageStamp
	stampMapDirty bool

	sync.RWMutex
}

type detailedMessage struct {
	Model *model.DetailedMessage

	stampMap      map[uuid.UUID]map[uuid.UUID]model.MessageStamp
	stampMapDirty bool

	sync.RWMutex
}

type quotedMessage struct {
	Model *model.QuotedMessage

	stampMap      map[uuid.UUID]map[uuid.UUID]model.MessageStamp
	stampMapDirty bool

	sync.RWMutex
}

func (m *message) GetID() uuid.UUID {
	m.RLock()
	defer m.RUnlock()
	return m.Model.ID
}

func (m *detailedMessage) GetID() uuid.UUID {
	m.RLock()
	defer m.RUnlock()
	return m.Model.ID
}

func (m *quotedMessage) GetID() uuid.UUID {
	m.RLock()
	defer m.RUnlock()
	return m.Model.ID
}

func (m *message) GetUserID() uuid.UUID {
	m.RLock()
	defer m.RUnlock()
	return m.Model.UserID
}

func (m *detailedMessage) GetUserID() uuid.UUID {
	m.RLock()
	defer m.RUnlock()
	return m.Model.UserID
}

func (m *quotedMessage) GetUserID() uuid.UUID {
	m.RLock()
	defer m.RUnlock()
	return m.Model.UserID
}

func (m *message) GetChannelID() uuid.UUID {
	m.RLock()
	defer m.RUnlock()
	return m.Model.ChannelID
}

func (m *detailedMessage) GetChannelID() uuid.UUID {
	m.RLock()
	defer m.RUnlock()
	return m.Model.ChannelID
}

func (m *quotedMessage) GetChannelID() uuid.UUID {
	m.RLock()
	defer m.RUnlock()
	return m.Model.ChannelID
}

func (m *message) GetText() string {
	m.RLock()
	defer m.RUnlock()
	return m.Model.Text
}

func (m *detailedMessage) GetText() string {
	m.RLock()
	defer m.RUnlock()
	return m.Model.Text
}

func (m *message) GetCreatedAt() time.Time {
	m.RLock()
	defer m.RUnlock()
	return m.Model.CreatedAt
}

func (m *detailedMessage) GetCreatedAt() time.Time {
	m.RLock()
	defer m.RUnlock()
	return m.Model.CreatedAt
}

func (m *quotedMessage) GetCreatedAt() time.Time {
	m.RLock()
	defer m.RUnlock()
	return m.Model.CreatedAt
}

func (m *message) GetUpdatedAt() time.Time {
	m.RLock()
	defer m.RUnlock()
	return m.Model.UpdatedAt
}

func (m *detailedMessage) GetUpdatedAt() time.Time {
	m.RLock()
	defer m.RUnlock()
	return m.Model.UpdatedAt
}

func (m *quotedMessage) GetUpdatedAt() time.Time {
	m.RLock()
	defer m.RUnlock()
	return m.Model.UpdatedAt
}

func (m *message) GetStamps() []model.MessageStamp {
	m.Lock()
	defer m.Unlock()
	if !m.stampMapDirty {
		return m.Model.Stamps
	}

	result := make([]model.MessageStamp, 0)
	for _, us := range m.stampMap {
		for _, ms := range us {
			result = append(result, ms)
		}
	}

	m.Model.Stamps = result
	m.stampMapDirty = false
	return result
}

func (m *detailedMessage) GetStamps() []model.MessageStamp {
	m.Lock()
	defer m.Unlock()
	if !m.stampMapDirty {
		return m.Model.Stamps
	}

	result := make([]model.MessageStamp, 0)
	for _, us := range m.stampMap {
		for _, ms := range us {
			result = append(result, ms)
		}
	}

	m.Model.Stamps = result
	m.stampMapDirty = false
	return result
}

func (m *quotedMessage) GetStamps() []model.MessageStamp {
	m.Lock()
	defer m.Unlock()
	if !m.stampMapDirty {
		return m.Model.Stamps
	}

	result := make([]model.MessageStamp, 0)
	for _, us := range m.stampMap {
		for _, ms := range us {
			result = append(result, ms)
		}
	}

	m.Model.Stamps = result
	m.stampMapDirty = false
	return result
}

func (m *detailedMessage) GetAttachments() []*model.FileMeta {
	m.RLock()
	defer m.RUnlock()
	return m.Model.Attachments
}

func (m *quotedMessage) GetAttachments() []*model.FileMeta {
	m.RLock()
	defer m.RUnlock()
	return m.Model.Attachments
}

func (m *detailedMessage) GetQuotes() []*model.QuotedMessage {
	m.RLock()
	defer m.RUnlock()
	return m.Model.Quotes
}

func (m *message) initStampMap() {
	m.stampMap = map[uuid.UUID]map[uuid.UUID]model.MessageStamp{}
	for _, ms := range m.Model.Stamps {
		m.addStamp(ms)
	}
}

func (m *message) addStamp(ms model.MessageStamp) {
	us, ok := m.stampMap[ms.StampID]
	if !ok {
		us = map[uuid.UUID]model.MessageStamp{}
		m.stampMap[ms.StampID] = us
	}
	us[ms.UserID] = ms
}

func (m *message) UpdateStamp(ms *model.MessageStamp) {
	m.Lock()
	defer m.Unlock()
	if m.stampMap == nil {
		m.initStampMap()
	}

	m.addStamp(*ms)
	m.stampMapDirty = true
}

func (m *message) RemoveStamp(stampID, userID uuid.UUID) {
	m.Lock()
	defer m.Unlock()
	if m.stampMap == nil {
		m.initStampMap()
	}

	us, ok := m.stampMap[stampID]
	if !ok {
		return
	}
	delete(us, userID)
	m.stampMapDirty = true
}

func (m *message) GetPin() *model.Pin {
	m.RLock()
	defer m.RUnlock()
	return m.Model.Pin
}

func (m *detailedMessage) GetPin() *model.Pin {
	m.RLock()
	defer m.RUnlock()
	return m.Model.Pin
}

func (m *quotedMessage) GetPin() *model.Pin {
	m.RLock()
	defer m.RUnlock()
	return m.Model.Pin
}

func (m *message) MarshalJSON() ([]byte, error) {
	type obj struct {
		ID        uuid.UUID              `json:"id"`
		UserID    uuid.UUID              `json:"userId"`
		ChannelID uuid.UUID              `json:"channelId"`
		Content   string                 `json:"content"`
		CreatedAt time.Time              `json:"createdAt"`
		UpdatedAt time.Time              `json:"updatedAt"`
		Pinned    bool                   `json:"pinned"`
		Stamps    []model.MessageStamp   `json:"stamps"`
		ThreadID  optional.Of[uuid.UUID] `json:"threadId"` // TODO
	}
	stamps := m.GetStamps()
	m.RLock()
	v := &obj{
		ID:        m.Model.ID,
		UserID:    m.Model.UserID,
		ChannelID: m.Model.ChannelID,
		Content:   m.Model.Text,
		CreatedAt: m.Model.CreatedAt,
		UpdatedAt: m.Model.UpdatedAt,
		Pinned:    m.Model.Pin != nil,
		Stamps:    stamps,
	}
	m.RUnlock()
	return jsonIter.ConfigFastest.Marshal(v)
}

func (m *detailedMessage) MarshalJSON() ([]byte, error) {
	type obj struct {
		ID          uuid.UUID              `json:"id"`
		UserID      uuid.UUID              `json:"userId"`
		ChannelID   uuid.UUID              `json:"channelId"`
		Content     string                 `json:"content"`
		CreatedAt   time.Time              `json:"createdAt"`
		UpdatedAt   time.Time              `json:"updatedAt"`
		Pinned      bool                   `json:"pinned"`
		Stamps      []model.MessageStamp   `json:"stamps"`
		ThreadID    optional.Of[uuid.UUID] `json:"threadId"` // TODO
		Attachments []*model.FileMeta      `json:"attachments"`
		Quotes      []*quotedMessage       `json:"quotes"`
	}
	stamps := m.GetStamps()
	m.RLock()
	quotes := make([]*quotedMessage, len(m.Model.Quotes))
	for i, q := range m.Model.Quotes {
		quotes[i] = &quotedMessage{Model: q}
	}
	v := &obj{
		ID:          m.Model.ID,
		UserID:      m.Model.UserID,
		ChannelID:   m.Model.ChannelID,
		Content:     m.Model.Text,
		CreatedAt:   m.Model.CreatedAt,
		UpdatedAt:   m.Model.UpdatedAt,
		Pinned:      m.Model.Pin != nil,
		Stamps:      stamps,
		Attachments: m.Model.Attachments,
		Quotes:      quotes,
	}
	m.RUnlock()
	return jsonIter.ConfigFastest.Marshal(v)
}

func (m *quotedMessage) MarshalJSON() ([]byte, error) {
	type obj struct {
		ID          uuid.UUID              `json:"id"`
		UserID      uuid.UUID              `json:"userId"`
		ChannelID   uuid.UUID              `json:"channelId"`
		Content     string                 `json:"content"`
		CreatedAt   time.Time              `json:"createdAt"`
		UpdatedAt   time.Time              `json:"updatedAt"`
		Pinned      bool                   `json:"pinned"`
		Stamps      []model.MessageStamp   `json:"stamps"`
		ThreadID    optional.Of[uuid.UUID] `json:"threadId"` // TODO
		Attachments []*model.FileMeta      `json:"attachments"`
	}
	stamps := m.GetStamps()
	m.RLock()
	v := &obj{
		ID:          m.Model.ID,
		UserID:      m.Model.UserID,
		ChannelID:   m.Model.ChannelID,
		Content:     m.Model.Text,
		CreatedAt:   m.Model.CreatedAt,
		UpdatedAt:   m.Model.UpdatedAt,
		Pinned:      m.Model.Pin != nil,
		Stamps:      stamps,
		Attachments: m.Model.Attachments,
	}
	m.RUnlock()
	return jsonIter.ConfigFastest.Marshal(v)
}
