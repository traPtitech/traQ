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
	// botも含む
	Get() int64
}

type userCounter struct {
	count int64
	sync.RWMutex
}
type usersCounterImpl struct {
	userCounter map[model.UserAccountStatus]*userCounter
	botCounter userCounter
	countersLock sync.Mutex
}
// NewUserCounter status別ユーザー数カウンタを生成します
func NewUserCounter(db *gorm.DB, hub *hub.Hub) (UserCounter, error) {
	counter := &usersCounterImpl{
		userCounter: make(map[model.UserAccountStatus]*userCounter),
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
	if err := counter.countBots(db); err != nil {
		return nil, err
	}
	
	usersCounter.WithLabelValues("deactivated").Set(float64(counter.userCounter[model.UserAccountStatusDeactivated].count))
	usersCounter.WithLabelValues("active").Set(float64(counter.userCounter[model.UserAccountStatusActive].count))
	usersCounter.WithLabelValues("suspended").Set(float64(counter.userCounter[model.UserAccountStatusSuspended].count))
	usersCounter.WithLabelValues("bot").Set(float64(counter.botCounter.count))

	return counter, nil
}

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
func (c *usersCounterImpl) countBots(db *gorm.DB) error {
	if err := db.
		Model(&model.User{}).
		Where("bot = ?", "1").
		Count(&c.botCounter.count).
		Error; err != nil {
		fmt.Errorf("failed to load Bots: %w", err)
	}
	return nil
}

func (c *usersCounterImpl) Get() int64 {
	c.countersLock.Lock()
	defer c.countersLock.Unlock()
	return c.userCounter[model.UserAccountStatusDeactivated].count + c.userCounter[model.UserAccountStatusActive].count + c.userCounter[model.UserAccountStatusSuspended].count + c.botCounter.count
}