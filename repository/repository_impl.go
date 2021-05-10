package repository

import (
	"github.com/leandro-lugaresi/hub"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/migration"
	"github.com/traPtitech/traQ/model"
)

// GormRepository リポジトリ実装
type GormRepository struct {
	db     *gorm.DB
	hub    *hub.Hub
	logger *zap.Logger
	stamps *stampRepository
}

// Sync implements Repository interface.
func (repo *GormRepository) Sync() (init bool, err error) {
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
func NewGormRepository(db *gorm.DB, hub *hub.Hub, logger *zap.Logger) (Repository, error) {
	repo := &GormRepository{
		db:     db,
		hub:    hub,
		logger: logger.Named("repository"),
	}
	return repo, nil
}
