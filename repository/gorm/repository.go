package gorm

import (
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/migration"
	"github.com/traPtitech/traQ/repository"
)

// Repository リポジトリ実装
type Repository struct {
	db     *gorm.DB
	hub    *hub.Hub
	logger *zap.Logger
	stamps *stampRepository
}

// NewGormRepository リポジトリ実装を初期化して生成します。
// スキーマが初期化された場合、init: true を返します。
func NewGormRepository(db *gorm.DB, hub *hub.Hub, logger *zap.Logger, doMigration bool) (repo repository.Repository, init bool, err error) {
	repo = &Repository{
		db:     db,
		hub:    hub,
		logger: logger.Named("repository"),
		stamps: makeStampRepository(db),
	}
	if doMigration {
		if init, err = migration.Migrate(db); err != nil {
			return nil, false, err
		}
	}
	return
}
