package migration

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/gofrs/uuid"
)

func v41() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "41",
		Migrate: func(db *gorm.DB) error {
			// No changes in this migration
			var groupRecords []v41GroupRecord
			if err := db.Table(v41GroupRecord{}.TableName()).Where("name REGEXP ?", "[@＠#＃:： 　]").Find(&groupRecords).Error; err != nil {
				return err
			}
			for _, record := range groupRecords {
				newName := record.Name
				newName = strings.ReplaceAll(newName, "@", "_")
				newName = strings.ReplaceAll(newName, "＠", "_")
				newName = strings.ReplaceAll(newName, "#", "_")
				newName = strings.ReplaceAll(newName, "＃", "_")
				newName = strings.ReplaceAll(newName, ":", "_")
				newName = strings.ReplaceAll(newName, "：", "_")
				newName = strings.ReplaceAll(newName, " ", "_")
				newName = strings.ReplaceAll(newName, "　", "_")
				var count int64
				if err := db.Table(v41GroupRecord{}.TableName()).Where("name = ?", newName).Model(&v41GroupRecord{}).Count(&count).Error; err != nil {
					return err
				}
				// 先頭15文字を残してランダムな英数字を付け加える
				for attempt := 0; count > 0 && attempt < 10; attempt++ {
					uniqueName := newName
					if len(uniqueName) > 15 {
						uniqueName = uniqueName[:15]
					}
					chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
					for len(uniqueName) < 30 {
						uniqueName += string(chars[rand.Intn(len(chars))])
					}
					if err := db.Table(v41GroupRecord{}.TableName()).Where("name = ?", uniqueName).Model(&v41GroupRecord{}).Count(&count).Error; err != nil {
						return err
					}
					if count == 0 {
						newName = uniqueName
						break
					}
				}
				if count > 0 {
					return fmt.Errorf("failed to generate a unique name for group %s after 10 attempts", record.Name)
				}
				// グループ名を書き換え
				if err := db.Table(v41GroupRecord{}.TableName()).Model(&v41GroupRecord{}).Where("id = ?", record.ID).Update("name", newName).Error; err != nil {
					return err
				}
			}
			return nil
		},
	}
}

type v41GroupRecord struct {
	ID          uuid.UUID `gorm:"type:char(36);not null;primaryKey"`
	Name        string    `gorm:"type:varchar(30);not null;unique"`
	Description string    `gorm:"type:text;not null"`
	Type        string    `gorm:"type:varchar(30);not null;default:''"`
	Icon        uuid.UUID `gorm:"type:char(36)"`
	CreatedAt   time.Time `gorm:"precision:6"`
	UpdatedAt   time.Time `gorm:"precision:6"`
}

func (v41GroupRecord) TableName() string {
	return "user_groups"
}
