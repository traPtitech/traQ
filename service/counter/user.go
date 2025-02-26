package counter

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/leandro-lugaresi/hub"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/event"
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
	userCounter      map[model.UserAccountStatus]*userCounter
	botCounter       map[model.BotState]*userCounter
	userTotalCounter userCounter
	botTotalCounter  userCounter
	countersLock     sync.Mutex
}

// NewUserCounter status別ユーザー数カウンタを生成します
func NewUserCounter(db *gorm.DB, hub *hub.Hub) (UserCounter, error) {
	counter := &usersCounterImpl{
		userCounter:      make(map[model.UserAccountStatus]*userCounter),
		botCounter:       make(map[model.BotState]*userCounter),
		userTotalCounter: userCounter{count: 0},
		botTotalCounter:  userCounter{count: 0},
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
	counter.userTotalCounter.count = counter.userCounter[model.UserAccountStatusDeactivated].count + counter.userCounter[model.UserAccountStatusActive].count + counter.userCounter[model.UserAccountStatusSuspended].count
	if err := counter.countBots(model.BotInactive, db); err != nil {
		return nil, err
	}
	if err := counter.countBots(model.BotActive, db); err != nil {
		return nil, err
	}
	if err := counter.countBots(model.BotPaused, db); err != nil {
		return nil, err
	}
	counter.botTotalCounter.count = counter.botCounter[model.BotInactive].count + counter.botCounter[model.BotActive].count + counter.botCounter[model.BotPaused].count
	usersCounter.WithLabelValues("user-deactivated").Set(float64(counter.userCounter[model.UserAccountStatusDeactivated].count))
	usersCounter.WithLabelValues("user-active").Set(float64(counter.userCounter[model.UserAccountStatusActive].count))
	usersCounter.WithLabelValues("user-suspended").Set(float64(counter.userCounter[model.UserAccountStatusSuspended].count))
	usersCounter.WithLabelValues("user-total").Set(float64(counter.userTotalCounter.count))
	usersCounter.WithLabelValues("bot-inactive").Set(float64(counter.botCounter[model.BotInactive].count))
	usersCounter.WithLabelValues("bot-active").Set(float64(counter.botCounter[model.BotActive].count))
	usersCounter.WithLabelValues("bot-paused").Set(float64(counter.botCounter[model.BotPaused].count))
	usersCounter.WithLabelValues("bot-total").Set(float64(counter.botTotalCounter.count))

	go func() {
		for e := range hub.Subscribe(1, event.UserCreated, event.UserUpdated, event.BotCreated, event.BotStateChanged, event.BotDeleted).Receiver {
			switch e.Topic() {
			case event.UserCreated:
				counter.userCounter[model.UserAccountStatusActive].inc("user-active")
				counter.userTotalCounter.inc("user-total")

			case event.UserUpdated:
				deacIsIncreased, deacIsDecreased := counter.userCounter[model.UserAccountStatusDeactivated].isChangedforUser("user-deactivated", model.UserAccountStatusDeactivated, db)
				susIsIncreased, susIsDecreased := counter.userCounter[model.UserAccountStatusSuspended].isChangedforUser("user-suspended", model.UserAccountStatusSuspended, db)
				if !deacIsIncreased && !susIsIncreased {
					counter.userCounter[model.UserAccountStatusActive].inc("user-active")
				}
				if !deacIsDecreased && !susIsDecreased {
					counter.userCounter[model.UserAccountStatusActive].dec("user-active")
				}

			case event.BotCreated:
				botState := e.Fields["bot"].(*model.Bot).State
				initialState := ""
				if botState == model.BotActive {
					initialState = "bot-active"
				} else if botState == model.BotInactive {
					initialState = "bot-inactive"
				}

				counter.botCounter[botState].inc(initialState)
				counter.userCounter[model.UserAccountStatusActive].dec("user-active")
				counter.botTotalCounter.inc("bot-total")
				counter.userTotalCounter.dec("user-total")

			case event.BotStateChanged:
				status2Label := map[model.BotState]string{
					model.BotInactive: "bot-inactive",
					model.BotActive:   "bot-active",
					model.BotPaused:   "bot-paused",
				}
				incStatus := e.Fields["state"].(model.BotState)
				incLabel := status2Label[incStatus]
				counter.botCounter[incStatus].inc(incLabel)

				nextStatus := model.BotState((int(incStatus) + 1) % 3)
				nextLabel := status2Label[nextStatus]
				_, isNextStatusDecreased := counter.botCounter[nextStatus].isChangedforBot(nextLabel, nextStatus, db)

				if !isNextStatusDecreased {
					lastStatus := model.BotState((int(incStatus) + 2) % 3)
					lastLabel := status2Label[lastStatus]
					counter.botCounter[lastStatus].dec(lastLabel)
				}

			case event.BotDeleted:
				counter.botTotalCounter.dec("bot-total")
				_, isDecreased := counter.botCounter[model.BotActive].isChangedforBot("bot-active", model.BotActive, db)
				if isDecreased {
					break
				}
				_, isDecreased = counter.botCounter[model.BotInactive].isChangedforBot("bot-inactive", model.BotInactive, db)
				if isDecreased {
					break
				}
				counter.botCounter[model.BotPaused].dec("bot-paused")
			}
		}
	}()
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
		return fmt.Errorf("failed to load Users: %w", err)
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
		return fmt.Errorf("failed to load Bots: %w", err)
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

// isChangedforUser 指定したステータスのユーザー数が変化したかを確認し、変化していた場合はカウンタを更新します
func (counter *userCounter) isChangedforUser(label string, userStatus model.UserAccountStatus, db *gorm.DB) (isIncreased, isDecreased bool) {
	var userCounter int64

	db.Model(&model.User{}).
		Where("bot = ?", "0").
		Where("status = ?", strconv.Itoa(userStatus.Int())).
		Count(&userCounter)

	if userCounter < counter.count {
		counter.dec(label)
		isIncreased = false
		isDecreased = true
		return
	}
	if userCounter > counter.count {
		counter.inc(label)
		isIncreased = true
		isDecreased = false
		return
	}
	isIncreased = false
	isDecreased = false
	return
}

// isChangedforBot 指定したステータスのBot数が変化したかを確認し、変化していた場合はカウンタを更新します
func (counter *userCounter) isChangedforBot(label string, botStatus model.BotState, db *gorm.DB) (isIncreased, isDecreased bool) {
	var userCounter int64

	db.Model(&model.Bot{}).
		Where("state = ?", strconv.Itoa(int(botStatus))).
		Count(&userCounter)

	if userCounter < counter.count {
		counter.dec(label)
		isIncreased = false
		isDecreased = true
		return
	} else if userCounter > counter.count {
		counter.inc(label)
		isIncreased = true
		isDecreased = false
		return
	}
	isIncreased = false
	isDecreased = false
	return
}

func (counter *userCounter) inc(label string) {
	counter.Lock()
	counter.count++
	counter.Unlock()
	usersCounter.WithLabelValues(label).Inc()
}
func (counter *userCounter) dec(label string) {
	counter.Lock()
	counter.count--
	counter.Unlock()
	usersCounter.WithLabelValues(label).Dec()
}
