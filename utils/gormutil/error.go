package gormutil

import (
	"github.com/go-sql-driver/mysql"
)

const (
	errMySQLDuplicatedRecord          uint16 = 1062
	errMySQLForeignKeyConstraintFails uint16 = 1452
)

// IsMySQLDuplicatedRecordErr MySQL重複レコードエラーかどうか
func IsMySQLDuplicatedRecordErr(err error) bool {
	mErr, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}
	return mErr.Number == errMySQLDuplicatedRecord
}

// IsMySQLForeignKeyConstraintFailsError MySQL外部キー制約エラーかどうか
func IsMySQLForeignKeyConstraintFailsError(err error) bool {
	mErr, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}
	return mErr.Number == errMySQLForeignKeyConstraintFails
}
