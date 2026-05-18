package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// v43 Fix user stamp history index
// Avoids filesort in GetUserStampRecommendations by covering (user_id, updated_at, stamp_id)
func v43() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "43",
		Migrate: func(db *gorm.DB) error {
			if err := db.Exec("DROP INDEX idx_messages_stamps_user_id_updated_at ON messages_stamps").Error; err != nil {
				return err
			}
			if err := db.Exec("CREATE INDEX idx_messages_stamps_user_id_updated_at_stamp_id ON messages_stamps (user_id, updated_at, stamp_id)").Error; err != nil {
				return err
			}
			return nil
		},
		Rollback: func(db *gorm.DB) error {
			if err := db.Exec("DROP INDEX idx_messages_stamps_user_id_updated_at_stamp_id ON messages_stamps").Error; err != nil {
				return err
			}
			if err := db.Exec("CREATE INDEX idx_messages_stamps_user_id_updated_at ON messages_stamps (user_id, updated_at)").Error; err != nil {
				return err
			}
			return nil
		},
	}
}
