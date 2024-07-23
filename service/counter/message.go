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
	initError      error
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
func NewMessageCounter(db *gorm.DB, hub *hub.Hub) (MessageCounter, error) {
	messageCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "traq",
		Name:      "messages_count_total",
	})
	mc := &messageCounterImpl{
		db:  db,
		hub: hub,
	}
	initOnce.Do(func() {
		initError = mc.initializeCounter()
	})
	if initError != nil {
		return nil, initError
	}
	return mc, nil
}

func (mc *messageCounterImpl) initializeCounter() error {
	var initialCount int64
	if err := mc.db.Unscoped().Model(&model.Message{}).Count(&initialCount).Error; err != nil {
		return err
	}
	atomic.StoreInt64(&count, initialCount)
	messageCounter.Add(float64(initialCount))

	go func() {
		for range mc.hub.Subscribe(1, event.MessageCreated).Receiver {
			mc.inc()
		}
	}()
	return nil
}

func (mc *messageCounterImpl) Get() int64 {
	initOnce.Do(func() {
		initError = mc.initializeCounter()
	})
	return atomic.LoadInt64(&count)
}

func (mc *messageCounterImpl) inc() {
	initOnce.Do(func() {
		initError = mc.initializeCounter()
	})
	atomic.AddInt64(&count, 1)
	messageCounter.Inc()
}
