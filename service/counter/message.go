package counter

import (
	"fmt"
	"sync"

	"github.com/leandro-lugaresi/hub"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
)

var messagesCounter = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "traq",
	Name:      "messages_count_total",
})

// MessageCounter 全メッセージ数カウンタ
type MessageCounter interface {
	// Get 全メッセージ数を返します
	//
	// この数値は削除されたメッセージを含んでいます
	Get() int64
}

type messageCounterImpl struct {
	count int64
	sync.RWMutex
}

// NewMessageCounter 全メッセージ数カウンタを生成します
func NewMessageCounter(db *gorm.DB, hub *hub.Hub) (MessageCounter, error) {
	counter := &messageCounterImpl{}
	if err := db.Unscoped().Model(&model.Message{}).Count(&counter.count).Error; err != nil {
		return nil, fmt.Errorf("failed to load total messages count: %w", err)
	}
	messagesCounter.Add(float64(counter.count))
	go func() {
		for range hub.Subscribe(1, event.MessageCreated).Receiver {
			counter.inc()
		}
	}()
	return counter, nil
}

func (c *messageCounterImpl) Get() int64 {
	c.RLock()
	defer c.RUnlock()
	return c.count
}

func (c *messageCounterImpl) inc() {
	c.Lock()
	c.count++
	c.Unlock()
	messagesCounter.Inc()
}
