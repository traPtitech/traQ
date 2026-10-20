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

var channelsCounter = promauto.NewCounter(prometheus.CounterOpts{
	Namespace: "traq",
	Name:      "channels_count_total",
})

// ChannelCounter 公開チャンネル数カウンタ
type ChannelCounter interface {
	// Get 公開チャンネル数を返します
	Get() int64
}

type channelCounterImpl struct {
	count int64
	sync.RWMutex
}

// NewChannelCounter 公開チャンネル数カウンタを生成します
func NewChannelCounter(db *gorm.DB, hub *hub.Hub) (ChannelCounter, error) {
	counter := &channelCounterImpl{}
	if err := db.Unscoped().Model(&model.Channel{}).Where(&model.Channel{IsPublic: true}).Count(&counter.count).Error; err != nil {
		return nil, fmt.Errorf("failed to load public channels count: %w", err)
	}
	channelsCounter.Add(float64(counter.count))
	go func() {
		for e := range hub.Subscribe(1, event.ChannelCreated).Receiver {
			if e.Fields["channel"].(*model.Channel).IsPublic {
				counter.inc()
			}
		}
	}()
	return counter, nil
}

func (c *channelCounterImpl) Get() int64 {
	c.RLock()
	defer c.RUnlock()
	return c.count
}

func (c *channelCounterImpl) inc() {
	c.Lock()
	c.count++
	c.Unlock()
	channelsCounter.Inc()
}
