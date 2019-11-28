package migration

import (
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/rbac/role"
	"gopkg.in/gormigrate.v1"
)

// データベースマイグレーション
var migrations = []*gormigrate.Migration{
	v1, // インデックスidx_messages_deleted_atの削除とidx_messages_channel_id_deleted_at_created_atの追加
	v2, // RBAC周りのリフォーム
	v3, // チャンネルイベント履歴
	v4, // Webhook, Bot外部キー
	v5, // Mute, 旧Clip削除
}

// Migrate データベースマイグレーションを実行します
func Migrate(db *gorm.DB) error {
	m := gormigrate.New(db, &gormigrate.Options{
		TableName:      "migrations",
		IDColumnName:   "id",
		IDColumnSize:   190,
		UseTransaction: false,
	}, migrations)
	m.InitSchema(func(db *gorm.DB) error {
		// 初回のみに呼ばれる
		// 全ての最新のデータベース定義を書く事

		// テーブル
		if err := db.AutoMigrate(AllTables...).Error; err != nil {
			return err
		}

		// 外部キー制約
		for _, c := range AllForeignKeys {
			if err := db.Table(c[0]).AddForeignKey(c[1], c[2], c[3], c[4]).Error; err != nil {
				return err
			}
		}

		// 複合インデックス
		for _, v := range AllCompositeIndexes {
			if err := db.Table(v[1]).AddIndex(v[0], v[2:]...).Error; err != nil {
				return err
			}
		}

		// 初期ユーザーロール投入
		for _, v := range role.SystemRoles() {
			if err := db.Create(v).Error; err != nil {
				return err
			}

			for _, v := range v.Permissions {
				if err := db.Create(v).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
	return m.Migrate()
}

// DropAll データベースの全テーブルを削除します
func DropAll(db *gorm.DB) error {
	if err := db.DropTableIfExists(AllTables...).Error; err != nil {
		return err
	}
	return db.DropTableIfExists("migrations").Error
}
