package counter

import (
	"sync"
	"sync/atomic"

	"github.com/leandro-lugaresi/hub"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
)

var (
	initOnce       sync.Once
	count          int64
	messageCounter prometheus.Counter
)

// MessageCounter 全メッセージ数カウンタ
type MessageCounter interface {
	// Get 全メッセージ数を返します
	//
	// この数値は削除されたメッセージを含んでいます
	Get() int64
}

type messageCounterImpl struct {
	db  *gorm.DB
	hub *hub.Hub
}

// NewMessageCounter 全メッセージ数カウンタを生成します
func NewMessageCounter(db *gorm.DB, hub *hub.Hub) MessageCounter {
	messageCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "traq",
		Name:      "messages_count_total",
	})
	mc := &messageCounterImpl{
		db:  db,
		hub: hub,
	}
	initOnce.Do(mc.initializeCounter)
	return mc
}

func (mc *messageCounterImpl) initializeCounter() {
	var initialCount int64
	if err := mc.db.Unscoped().Model(&model.Message{}).Count(&initialCount).Error; err != nil {
		panic(err)
	}
	atomic.StoreInt64(&count, initialCount)
	messageCounter.Add(float64(initialCount))

	go func() {
		for range mc.hub.Subscribe(1, event.MessageCreated).Receiver {
			mc.inc()
		}
	}()
}

func (mc *messageCounterImpl) Get() int64 {
	initOnce.Do(mc.initializeCounter)
	return atomic.LoadInt64(&count)
}

func (mc *messageCounterImpl) inc() {
	initOnce.Do(mc.initializeCounter)
	atomic.AddInt64(&count, 1)
	messageCounter.Inc()
}
