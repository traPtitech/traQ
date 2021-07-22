package counter

import (
	"fmt"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
)

// UnreadMessageCounter 未読メッセージ数カウンタ
type UnreadMessageCounter interface {
	// Get 指定したユーザーの未読メッセージ数を返します
	Get(userID uuid.UUID) int
	// GetChanges 未読メッセージ数が変化したユーザーの未読メッセージ数を返します。
	//
	// resetをtrueにすると変化の記録をリセットします
	GetChanges(reset bool) map[uuid.UUID]int
}

type unreadMessageCounterImpl struct {
	counters map[uuid.UUID]int
	changed  map[uuid.UUID]struct{}
	mu       sync.RWMutex
}

// NewUnreadMessageCounter 未読メッセージ数カウンタを生成します
func NewUnreadMessageCounter(db *gorm.DB, hub *hub.Hub) (UnreadMessageCounter, error) {
	type count struct {
		UserID uuid.UUID
		Count  int
	}
	var counts []*count
	if err := db.Raw(`SELECT user_id, COUNT(user_id) AS count FROM unreads GROUP BY user_id`).Scan(&counts).Error; err != nil {
		return nil, fmt.Errorf("failed to load unread messages count: %w", err)
	}
	impl := &unreadMessageCounterImpl{
		counters: make(map[uuid.UUID]int, len(counts)),
		changed:  make(map[uuid.UUID]struct{}, len(counts)),
	}
	for _, c := range counts {
		impl.counters[c.UserID] = c.Count
		impl.changed[c.UserID] = struct{}{}
	}

	go func() {
		for e := range hub.Subscribe(8, event.MessageUnread, event.ChannelRead, event.MessageDeleted).Receiver {
			switch e.Topic() {
			case event.MessageUnread:
				impl.Inc(e.Fields["user_id"].(uuid.UUID), 1)
			case event.ChannelRead:
				impl.Dec(e.Fields["user_id"].(uuid.UUID), e.Fields["read_messages_num"].(int))
			case event.MessageDeleted:
				impl.DecMultiple(e.Fields["deleted_unreads"].([]*model.Unread))
			}
		}
	}()
	return impl, nil
}

func (c *unreadMessageCounterImpl) Inc(userID uuid.UUID, n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counters[userID] += n
	c.changed[userID] = struct{}{}
}

func (c *unreadMessageCounterImpl) Get(userID uuid.UUID) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.counters[userID]
}

func (c *unreadMessageCounterImpl) GetChanges(reset bool) map[uuid.UUID]int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	changes := make(map[uuid.UUID]int, len(c.changed))
	for id := range c.changed {
		changes[id] = c.counters[id]
	}
	if reset {
		c.changed = make(map[uuid.UUID]struct{}, len(c.counters))
	}
	return changes
}

func (c *unreadMessageCounterImpl) Dec(userID uuid.UUID, n int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dec(userID, n)
}

func (c *unreadMessageCounterImpl) DecMultiple(unreads []*model.Unread) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, unread := range unreads {
		c.dec(unread.UserID, 1)
	}
}

func (c *unreadMessageCounterImpl) dec(userID uuid.UUID, n int) {
	c.changed[userID] = struct{}{}
	result := c.counters[userID] - n
	if result <= 0 {
		delete(c.counters, userID)
		return
	}
	c.counters[userID] = result
}
