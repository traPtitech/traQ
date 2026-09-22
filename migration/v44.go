package migration

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// スタンプ名の接頭が"0x"であるときにスタンプ名を置換するマイグレーション
func v44() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "44",
		Migrate: func(db *gorm.DB) error {
			var stamps []struct {
				ID   uuid.UUID
				Name string
			}

			// SELECT FROM stamp WHERE name LIKE BINARY 0x% ORDER BY id
			if err := db.Table("stamps").Where("name LIKE BINARY ?", "0x%").Order("id").Find(&stamps).Error; err != nil {
				return err
			}

			//スタンプ名の置換
			for _, stamp := range stamps {
				runes := []rune(stamp.Name)

				//"0x%"を"_x%"に変換して、新しい名前として仮決定
				runes[0] = '_'

				newName := string(runes)

				// 重複しなくなるまで末尾に接尾辞("_1", "_2"など)を追加
				suffixIndex := 1
				for {
					var count int64
					if err := db.Table("stamps").Where("name = ?", newName).Count(&count).Error; err != nil {
						return err
					}
					if count == 0 {
						break
					}

					//接尾辞("_1"など)を作成
					suffix := fmt.Sprintf("_%d", suffixIndex)

					//語数がオーバーした場合は元の名前の後ろを切り詰める
					maxNameLen := 32

					trimLen := maxNameLen - len([]rune(suffix))
					if len(runes) > trimLen {
						newName = string(runes[:trimLen]) + suffix
					} else {
						newName = string(runes) + suffix
					}

					suffixIndex++
				}

				//スタンプ名を更新
				if err := db.Table("stamps").Where("id = ?", stamp.ID).Update("name", newName).Error; err != nil {
					return err
				}
			}

			return nil
		},
	}
}
