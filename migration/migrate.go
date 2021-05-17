package migration

import (
	"database/sql"
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// Migrate データベースマイグレーションを実行します
// 初回実行でスキーマが初期化された場合、initでtrueを返します
func Migrate(db *gorm.DB) (init bool, err error) {
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
		init = true

		// テーブル
		return db.AutoMigrate(AllTables()...)
	})
	err = m.Migrate()
	return
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
