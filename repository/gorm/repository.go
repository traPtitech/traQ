package gorm

import (
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/migration"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

// Repository リポジトリ実装
type Repository struct {
	db     *gorm.DB
	hub    *hub.Hub
	logger *zap.Logger
	stamps *stampRepository
}

// Sync implements Repository interface.
func (repo *Repository) Sync() (init bool, err error) {
	if init, err = migration.Migrate(repo.db); err != nil {
		return false, err
	}

	// スタンプをキャッシュ
	var stamps []*model.Stamp
	if err := repo.db.Find(&stamps).Error; err != nil {
		return false, err
	}
	repo.stamps = makeStampRepository(stamps)

	return
}

// NewGormRepository リポジトリ実装を初期化して生成します
func NewGormRepository(db *gorm.DB, hub *hub.Hub, logger *zap.Logger) (repository.Repository, error) {
	repo := &Repository{
		db:     db,
		hub:    hub,
		logger: logger.Named("repository"),
	}
	return repo, nil
}
