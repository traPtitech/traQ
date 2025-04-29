package counter

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/gofrs/uuid"
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

// botStatus: keyのuuidはmodel.BotのIDです
type usersCounterImpl struct {
	userCounter      map[model.UserAccountStatus]*userCounter
	botCounter       map[model.BotState]*userCounter
	userTotalCounter userCounter
	botTotalCounter  userCounter
	userStatus       map[uuid.UUID]model.UserAccountStatus
	botStatus        map[uuid.UUID]model.BotState
	countersLock     sync.Mutex
}

// NewUserCounter status別ユーザー数カウンタを生成します
func NewUserCounter(db *gorm.DB, hub *hub.Hub) (UserCounter, error) {
	counter := &usersCounterImpl{
		userCounter:      make(map[model.UserAccountStatus]*userCounter),
		botCounter:       make(map[model.BotState]*userCounter),
		userTotalCounter: userCounter{count: 0},
		botTotalCounter:  userCounter{count: 0},
		userStatus:       make(map[uuid.UUID]model.UserAccountStatus),
		botStatus:        make(map[uuid.UUID]model.BotState),
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

	userStatus2Label := map[model.UserAccountStatus]string{
		model.UserAccountStatusDeactivated: "user-deactivated",
		model.UserAccountStatusActive:      "user-active",
		model.UserAccountStatusSuspended:   "user-suspended",
	}
	botStatus2Label := map[model.BotState]string{
		model.BotInactive: "bot-inactive",
		model.BotActive:   "bot-active",
		model.BotPaused:   "bot-paused",
	}
	go func() {
		for e := range hub.Subscribe(1, event.UserCreated, event.UserUpdated, event.BotCreated, event.BotStateChanged, event.BotDeleted).Receiver {
			switch e.Topic() {
			case event.UserCreated:
				createdUser := e.Fields["user"].(*model.User)
				if createdUser.Bot {
					break
				}
				counter.userCounter[model.UserAccountStatusActive].inc()
				counter.userTotalCounter.inc()
				usersCounter.WithLabelValues("user-active").Inc()
				usersCounter.WithLabelValues("user-total").Inc()
				counter.userStatus[createdUser.ID] = model.UserAccountStatusActive

			case event.UserUpdated:
				userID := e.Fields["user_id"].(uuid.UUID)
				var newStatusInInt int
				db.Model(&model.User{}).Where("id = ?", userID).Select("status").First(&newStatusInInt)
				newStatus := model.UserAccountStatus(newStatusInInt)
				preStatus := counter.userStatus[userID]

				counter.userCounter[preStatus].dec()
				counter.userCounter[newStatus].inc()
				usersCounter.WithLabelValues(userStatus2Label[preStatus]).Dec()
				usersCounter.WithLabelValues(userStatus2Label[newStatus]).Inc()
				counter.userStatus[userID] = newStatus

			case event.BotCreated:
				botState := e.Fields["bot"].(*model.Bot).State

				counter.botCounter[botState].inc()
				counter.botTotalCounter.inc()
				usersCounter.WithLabelValues(botStatus2Label[botState]).Inc()
				usersCounter.WithLabelValues("bot-total").Inc()
				createdBotID := e.Fields["bot_id"].(uuid.UUID)
				counter.botStatus[createdBotID] = botState


			case event.BotStateChanged:
				changedBotID := e.Fields["bot_id"].(uuid.UUID)
				newStatus := e.Fields["state"].(model.BotState)
				preStatus := counter.botStatus[changedBotID]

				counter.botCounter[preStatus].dec()
				counter.botCounter[newStatus].inc()
				usersCounter.WithLabelValues(botStatus2Label[preStatus]).Dec()
				usersCounter.WithLabelValues(botStatus2Label[newStatus]).Inc()
				counter.botStatus[changedBotID] = newStatus

			case event.BotDeleted:
				deletedBotID := e.Fields["bot_id"].(uuid.UUID)
				deletedBotState := counter.botStatus[deletedBotID]

				counter.botCounter[deletedBotState].dec()
				counter.botTotalCounter.dec()
				usersCounter.WithLabelValues(botStatus2Label[deletedBotState]).Dec()
				usersCounter.WithLabelValues("bot-total").Dec()
				delete(counter.botStatus, deletedBotID)
			}
		}
	}()
	return counter, nil
}

// countUsers 指定したステータスのユーザーを取得し、c.userStatusに保存、c.userCounterにカウントします
func (c *usersCounterImpl) countUsers(status model.UserAccountStatus, db *gorm.DB) error {
	tmpUserCounter := userCounter{
		count: 0,
	}
	var selectedStateUser []model.User
	if err := db.
		Model(&model.User{}).
		Where("bot = ?", "0").
		Where("status = ?", strconv.Itoa(status.Int())).
		Select("id").
		Find(&selectedStateUser).
		Error; err != nil {
		return fmt.Errorf("failed to load Users: %w", err)
	}

	for _, user := range selectedStateUser {
		c.userStatus[user.ID] = status
	}
	tmpUserCounter.count = int64(len(selectedStateUser))
	c.userCounter[status] = &tmpUserCounter
	return nil
}

// countBots 指定したステータスのBotを取得し、c.botStatusに保存、c.botCounterにカウントします
func (c *usersCounterImpl) countBots(status model.BotState, db *gorm.DB) error {
	tmpUserCounter := userCounter{
		count: 0,
	}
	var selectedStateBots []model.Bot
	if err := db.
		Model(&model.Bot{}).
		Where("state = ?", strconv.Itoa(int(status))).
		Select("id").
		Find(&selectedStateBots).
		Error; err != nil {
		return fmt.Errorf("failed to load Bots: %w", err)
	}

	for _, bot := range selectedStateBots {
		c.botStatus[bot.ID] = status
	}
	tmpUserCounter.count = int64(len(selectedStateBots))
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

func (counter *userCounter) inc() {
	counter.Lock()
	counter.count++
	counter.Unlock()
}
func (counter *userCounter) dec() {
	counter.Lock()
	counter.count--
	counter.Unlock()
}
