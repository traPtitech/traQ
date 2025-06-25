package migration

import (
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/gofrs/uuid"
)

func v40() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "40",
		Migrate: func(db *gorm.DB) error {
			// No changes in this migration
			var groupRecords []v40GroupRecord
			db.Table(v40GroupRecord{}.TableName()).Find(&groupRecords).Where("name REGEXP ^[^@＠#＃:： 　]*$")
			for _, record := range groupRecords {
				newName := record.Name
				newName = strings.ReplaceAll(record.Name, "@", "_")
				newName = strings.ReplaceAll(newName, "＠", "_")
				newName = strings.ReplaceAll(newName, "#", "_")
				newName = strings.ReplaceAll(newName, "＃", "_")
				newName = strings.ReplaceAll(newName, ":", "_")
				newName = strings.ReplaceAll(newName, "：", "_")
				newName = strings.ReplaceAll(newName, " ", "_")
				newName = strings.ReplaceAll(newName, "　", "_")
				var count int64
				if err := db.Table(v40GroupRecord{}.TableName()).Where("name = ?", newName).Model(&v40GroupRecord{}).Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					newName = strconv.Itoa(rand.Intn(999999999999999))
				}
				db.Table(v40GroupRecord{}.TableName()).Model(&v40GroupRecord{}).Where("id = ?", record.ID).Update("name", newName)
			}
			return nil
		},
	}
}

type v40GroupRecord struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Name        string    `gorm:"type:varchar(30);not null;unique"`
	Description string    `gorm:"type:text;not null"`
	Type        string    `gorm:"type:varchar(30);not null;default:''"`
	Icon        uuid.UUID `gorm:"type:char(36)"`
	CreatedAt   time.Time `gorm:"precision:6"`
	UpdatedAt   time.Time `gorm:"precision:6"`
}

func (v40GroupRecord) TableName() string {
	return "user_groups"
}
