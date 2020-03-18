package model

import (
	"github.com/gofrs/uuid"
	"time"
)

type StampPalettes struct {
	ID          uuid.UUID  `gorm:"type:char(36);not null;primary_key"`
	Name        string     `gorm:"type:varchar(30);not null"`
	Description string     `gorm:"type:varchar(300)"`
	CreatorID   uuid.UUID  `gorm:"type:char(36);not null"`
	CreatedAt   time.Time  `gorm:"precision:6"`
	DeletedAt   *time.Time `gorm:"precision:6"`
}
