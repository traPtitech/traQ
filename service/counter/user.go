package counter

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/leandro-lugaresi/hub"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/model"
)

var usersCounter = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "traq",
		Name:      "users_count",
	}, []string{"user_type"})

// UserCounter status別ユーザー数カウンタ
type UserCounter interface {
	// Get 全ユーザー数を返します
	// 削除されていないbotも含みます
	Get() int64
}

type userCounter struct {
	count int64
	sync.RWMutex
}
type usersCounterImpl struct {
	userCounter map[model.UserAccountStatus]*userCounter
	botCounter map[model.BotState]*userCounter
	countersLock sync.Mutex
}
// NewUserCounter status別ユーザー数カウンタを生成します
func NewUserCounter(db *gorm.DB, hub *hub.Hub) (UserCounter, error) {
	counter := &usersCounterImpl{
		userCounter: make(map[model.UserAccountStatus]*userCounter),
		botCounter: make(map[model.BotState]*userCounter),
	}

	if err := counter.countUsers(model.UserAccountStatusDeactivated, db); err != nil {
		return nil, err
	}
	if err := counter.countUsers(model.UserAccountStatusActive, db); err != nil {
		return nil, err
	}
	if err := counter.countUsers(model.UserAccountStatusSuspended, db); err != nil {
		return nil, err
	}
	if err := counter.countBots(model.BotInactive, db); err != nil {
		return nil, err
	}
	if err := counter.countBots(model.BotActive, db); err != nil {
		return nil, err
	}
	if err := counter.countBots(model.BotPaused, db); err != nil {
		return nil, err
	}
	usersCounter.WithLabelValues("user-deactivated").Set(float64(counter.userCounter[model.UserAccountStatusDeactivated].count))
	usersCounter.WithLabelValues("user-active").Set(float64(counter.userCounter[model.UserAccountStatusActive].count))
	usersCounter.WithLabelValues("user-suspended").Set(float64(counter.userCounter[model.UserAccountStatusSuspended].count))
	usersCounter.WithLabelValues("bot-inactive").Set(float64(counter.botCounter[model.BotInactive].count))
	usersCounter.WithLabelValues("bot-active").Set(float64(counter.botCounter[model.BotActive].count))
	usersCounter.WithLabelValues("bot-paused").Set(float64(counter.botCounter[model.BotPaused].count))

	return counter, nil
}

// countUsers 指定したステータスのユーザー数をカウントします
func (c *usersCounterImpl) countUsers(status model.UserAccountStatus, db *gorm.DB) error {
	tmpUserCounter := userCounter{
		count: 0,
	}
	if err := db.
		Model(&model.User{}).
		Where("bot = ?", "0").
		Where("status = ?", strconv.Itoa(status.Int())).
		Count(&tmpUserCounter.count).
		Error; err != nil {
		fmt.Errorf("failed to load Users: %w", err)
	}
	c.userCounter[status] = &tmpUserCounter
	return nil
}

// countBots 指定したステータスのBot数をカウントします
func (c *usersCounterImpl) countBots(status model.BotState, db *gorm.DB) error {
	tmpUserCounter := userCounter{
		count: 0,
	}
	if err := db.
		Model(&model.Bot{}).
		Where("state = ?", strconv.Itoa(int(status))).
		Count(&tmpUserCounter.count).
		Error; err != nil {
		fmt.Errorf("failed to load Bots: %w", err)
	}
	c.botCounter[status] = &tmpUserCounter
	return nil
}

// Get 全ユーザー数を返します
// botも含む
func (c *usersCounterImpl) Get() int64 {
	c.countersLock.Lock()
	defer c.countersLock.Unlock()
	return c.userCounter[model.UserAccountStatusDeactivated].count + c.userCounter[model.UserAccountStatusActive].count + c.userCounter[model.UserAccountStatusSuspended].count + 
		c.botCounter[model.BotInactive].count + c.botCounter[model.BotActive].count + c.botCounter[model.BotPaused].count
}