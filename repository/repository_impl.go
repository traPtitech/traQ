package repository

import (
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/utils/storage"
	"time"
)

const (
	errMySQLDuplicatedRecord uint16 = 1062
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
	onlineUsersCounter = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "traq",
		Name:      "online_users",
	})
)

// GormRepository リポジトリ実装
type GormRepository struct {
	db  *gorm.DB
	hub *hub.Hub
	channelImpl
	fileImpl
	*heartbeatImpl
}

// Sync implements Repository interface.
func (repo *GormRepository) Sync() (bool, error) {
	// スキーマ同期
	if err := repo.db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4").AutoMigrate(model.Tables...).Error; err != nil {
		return false, fmt.Errorf("failed to sync Table schema: %v", err)
	}

	// 外部キー制約同期
	for _, c := range model.Constraints {
		if err := repo.db.Table(c[0]).AddForeignKey(c[1], c[2], c[3], c[4]).Error; err != nil {
			return false, err
		}
	}

	// サーバーユーザーの確認
	c := 0
	err := repo.db.Model(&model.User{}).Where(&model.User{Role: role.Admin.ID()}).Limit(1).Count(&c).Error
	if err != nil {
		return false, err
	}
	if c == 0 {
		_, err := repo.CreateUser("traq", "traq", role.Admin)
		if err != nil {
			return false, err
		}
		return true, err
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
	return false, nil
}

// GetFS implements Repository interface.
func (repo *GormRepository) GetFS() storage.FileStorage {
	return repo.FS
}

// NewGormRepository リポジトリ実装を初期化して生成します
func NewGormRepository(db *gorm.DB, fs storage.FileStorage, hub *hub.Hub) (Repository, error) {
	repo := &GormRepository{
		db:  db,
		hub: hub,
		fileImpl: fileImpl{
			FS: fs,
		},
		heartbeatImpl: newHeartbeatImpl(hub),
	}
	go func() {
		sub := hub.Subscribe(10, event.UserOffline)
		for ev := range sub.Receiver {
			userID := ev.Fields["user_id"].(uuid.UUID)
			datetime := ev.Fields["datetime"].(time.Time)
			_ = repo.UpdateUserLastOnline(userID, datetime)
		}
	}()
	return repo, nil
}

func (repo *GormRepository) transact(txFunc func(tx *gorm.DB) error) (err error) {
	tx := repo.db.Begin()
	if err := tx.Error; err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic after Rollback
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit().Error
		}
	}()
	err = txFunc(tx)
	return err
}

func limitAndOffset(limit, offset int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if offset > 0 {
			db = db.Offset(offset)
		}
		if limit > 0 {
			db = db.Limit(limit)
		}
		return db
	}
}

func isMySQLDuplicatedRecordErr(err error) bool {
	merr, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}
	return merr.Number == errMySQLDuplicatedRecord
}

func dbExists(tx *gorm.DB, where interface{}, tableName ...string) (exists bool, err error) {
	c := 0
	if len(tableName) > 0 {
		tx = tx.Table(tableName[0])
	} else {
		tx = tx.Model(where)
	}
	err = tx.Where(where).Limit(1).Count(&c).Error
	return c > 0, err
}

func convertError(err error) error {
	switch {
	case gorm.IsRecordNotFoundError(err):
		return ErrNotFound
	default:
		return err
	}
}
