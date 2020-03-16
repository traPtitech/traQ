package migration

import (
	"database/sql"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/traPtitech/traQ/rbac/role"
	"gopkg.in/gormigrate.v1"
)

// Migrate データベースマイグレーションを実行します
func Migrate(db *gorm.DB) error {
	// データベースマイグレーション
	migrations := []*gormigrate.Migration{
		v1(), // インデックスidx_messages_deleted_atの削除とidx_messages_channel_id_deleted_at_created_atの追加
		v2(), // RBAC周りのリフォーム
		v3(), // チャンネルイベント履歴
		v4(), // Webhook, Bot外部キー
		v5(), // Mute, 旧Clip削除
		v6(), // v6 ユーザーグループ拡張
		v7(), // ファイルメタ拡張
		v8(), // チャンネル購読拡張
		v9(), // ユーザーテーブル拡張
	}

	m := gormigrate.New(db.Set("gorm:table_options", "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"), &gormigrate.Options{
		TableName:      "migrations",
		IDColumnName:   "id",
		IDColumnSize:   190,
		UseTransaction: false,
	}, migrations)
	m.InitSchema(func(db *gorm.DB) error {
		// 初回のみに呼ばれる
		// 全ての最新のデータベース定義を書く事

		// テーブル
		if err := db.AutoMigrate(AllTables()...).Error; err != nil {
			return err
		}

		// 外部キー制約
		for _, c := range AllForeignKeys() {
			if err := db.Table(c[0]).AddForeignKey(c[1], c[2], c[3], c[4]).Error; err != nil {
				return err
			}
		}

		// 複合インデックス
		for _, v := range AllCompositeIndexes() {
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
	if err := db.DropTableIfExists(AllTables()...).Error; err != nil {
		return err
	}
	return db.DropTableIfExists("migrations").Error
}

// CreateDatabasesIfNotExists データベースが存在しなければ作成します
func CreateDatabasesIfNotExists(dialect, dsn, prefix string, names ...string) error {
	conn, err := sql.Open(dialect, dsn)
	if err != nil {
		return err
	}
	defer conn.Close()
	for _, v := range names {
		_, err = conn.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s%s`", prefix, v))
		if err != nil {
			return err
		}
	}
	return nil
}
