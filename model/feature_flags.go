package model

import (
	"github.com/gofrs/uuid"
)

// FeatureFlag FeatureFlagの管理をする構造体
type FeatureFlag struct {
	UserID           uuid.UUID `gorm:"type:char(36);not null;primaryKey;"`
	FeatureFlagsJSON string    `gorm:"type:TEXT NOT NULL"`
}

// TableName DBの名前を指定
func (f *FeatureFlag) TableName() string {
	return "feature_flags"
}
