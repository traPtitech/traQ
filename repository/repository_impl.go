package repository

import (
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/migration"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/storage"
	"go.uber.org/zap"
	"time"
)

var (
	initialized     = false
	messagesCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "traq",
		Name:      "messages_count_total",
	})
	channelsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "traq",
		Name:      "channels_count_total",
	})
)

// GormRepository リポジトリ実装
type GormRepository struct {
	db     *gorm.DB
	hub    *hub.Hub
	logger *zap.Logger
	chTree *channelTreeImpl
	fileImpl
}

// Channel implements ReplaceMapper interface.
func (repo *GormRepository) Channel(path string) (uuid.UUID, bool) {
	id := repo.chTree.GetChannelIDFromPath(path)
	return id, id != uuid.Nil
}

// Group implements ReplaceMapper interface.
func (repo *GormRepository) Group(name string) (uuid.UUID, bool) {
	g, err := repo.GetUserGroupByName(name)
	if err != nil {
		return uuid.Nil, false
	}
	return g.ID, true
}

// User implements ReplaceMapper interface.
func (repo *GormRepository) User(name string) (uuid.UUID, bool) {
	u, err := repo.GetUserByName(name, false)
	if err != nil {
		return uuid.Nil, false
	}
	return u.GetID(), true
}

// Sync implements Repository interface.
func (repo *GormRepository) Sync() (init bool, err error) {
	if err := migration.Migrate(repo.db); err != nil {
		return false, err
	}

	// チャンネルツリー構築
	var channels []*model.Channel
	if err := repo.db.Where(&model.Channel{IsPublic: true}).Find(&channels).Error; err != nil {
		return false, err
	}
	repo.chTree, err = makeChannelTree(channels)
	if err != nil {
		return false, err
	}

	// メトリクス用データ取得
	if !initialized {
		initialized = true

		messageNum := 0
		if err := repo.db.Unscoped().Model(&model.Message{}).Count(&messageNum).Error; err != nil {
			return false, err
		}
		messagesCounter.Add(float64(messageNum))

		channelNum := 0
		if err := repo.db.Unscoped().Model(&model.Channel{}).Where(&model.Channel{IsPublic: true}).Count(&channelNum).Error; err != nil {
			return false, err
		}
		channelsCounter.Add(float64(channelNum))
	}

	// 管理者ユーザーの確認
	c := 0
	if err := repo.db.Model(&model.User{}).Where(&model.User{Role: role.Admin}).Limit(1).Count(&c).Error; err != nil {
		return false, err
	}
	if c == 0 {
		_, err := repo.CreateUser(CreateUserArgs{
			Name:     "traq",
			Password: "traq",
			Role:     role.Admin,
		})
		if err != nil {
			return false, err
		}
		return true, err
	}

	return false, nil
}

// GetFS implements Repository interface.
func (repo *GormRepository) GetFS() storage.FileStorage {
	return repo.FS
}

// NewGormRepository リポジトリ実装を初期化して生成します
func NewGormRepository(db *gorm.DB, fs storage.FileStorage, hub *hub.Hub, logger *zap.Logger) (Repository, error) {
	repo := &GormRepository{
		db:     db,
		hub:    hub,
		logger: logger,
		fileImpl: fileImpl{
			FS: fs,
		},
	}
	go func() {
		sub := hub.Subscribe(10, event.UserOffline)
		for ev := range sub.Receiver {
			userID := ev.Fields["user_id"].(uuid.UUID)
			datetime := ev.Fields["datetime"].(time.Time)
			_ = repo.UpdateUser(userID, UpdateUserArgs{LastOnline: optional.TimeFrom(datetime)})
		}
	}()
	return repo, nil
}
