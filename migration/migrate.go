package migration

import (
	"database/sql"
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/service/rbac/role"
)

// Migrate データベースマイグレーションを実行します
func Migrate(db *gorm.DB) error {
	m := gormigrate.New(db, &gormigrate.Options{
		TableName:                 "migrations",
		IDColumnName:              "id",
		IDColumnSize:              190,
		UseTransaction:            false,
		ValidateUnknownMigrations: true,
	}, Migrations())
	m.InitSchema(func(db *gorm.DB) error {
		// 初回のみに呼ばれる
		// 全ての最新のデータベース定義を書く事

		// テーブル
		if err := db.AutoMigrate(AllTables()...); err != nil {
			return err
		}

		// 初期ユーザーロール投入
		// (user_role, user_role_permissions, user_role_inheritances 作成)
		return db.Create(role.SystemRoleModels()).Error
	})
	return m.Migrate()
}

// DropAll データベースの全テーブルを削除します
func DropAll(db *gorm.DB) error {
	if err := db.Migrator().DropTable(AllTables()...); err != nil {
		return err
	}
	return db.Migrator().DropTable("migrations")
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
