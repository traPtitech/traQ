// Package migration では、gopkg.in/gormigrate.v2を用いたデータベースマイグレーションコードを記述します。
// データベースに新たなテーブルの追加や、既にあるテーブルのカラムの型の変更などのスキーマの改変を行う場合、必ずその改変処理(SQLクエリ)を全てここに記述してください。
// サーバー起動時に自動的にマイグレーションが実行されます。
//
// # Instruction 記述方法
//
// 既にv**.goに記述されているように、IDをインクリメントして*gormigrate.Migrationを返す関数を生成し、current.goにあるMigrationsに実装したマイグレーションを追加してください。
// 更に、全てのマイグレーションを適用後の最新のデータベーススキーマに関する情報を、AllTablesと対応する構造体のタグを変更することによって記述してください。
//
// v6.goの様に、新たに作成するテーブルや、古いバージョンのテーブルの情報を新しいテーブルに移行させる時に使う構造体は、modelパッケージに定義する・されている構造体を使うのではなく、
// v6UserGroupAdmin、v6OldUserGroupなどの様に、マイグレーションコード内に再定義したものを使ってください。
package migration
