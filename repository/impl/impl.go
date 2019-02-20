package impl

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/leandro-lugaresi/hub"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/storage"
	"time"
)

// RepositoryImpl リポジトリ実装
type RepositoryImpl struct {
	db  *gorm.DB
	hub *hub.Hub
	channelImpl
	fileImpl
	*heartbeatImpl
}

// Sync DBと同期します
func (repo *RepositoryImpl) Sync() (bool, error) {
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
	err := repo.db.Model(&model.User{}).Where(&model.User{Name: "traq"}).Limit(1).Count(&c).Error
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
	return false, nil
}

// NewRepositoryImpl リポジトリ実装を初期化して生成します
func NewRepositoryImpl(db *gorm.DB, fs storage.FileStorage, hub *hub.Hub) (repository.Repository, error) {
	repo := &RepositoryImpl{
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

func (repo *RepositoryImpl) transact(txFunc func(tx *gorm.DB) error) (err error) {
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
