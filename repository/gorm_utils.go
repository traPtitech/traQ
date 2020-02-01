package repository

import (
	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

const (
	errMySQLDuplicatedRecord          uint16 = 1062
	errMySQLForeignKeyConstraintFails uint16 = 1452
)

func limitAndOffset(limit, offset int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if offset > 0 {
			db = db.Offset(offset)
		}
		if limit > 0 {
			db = db.Limit(limit)
		}
		return db
	}
}

func isMySQLDuplicatedRecordErr(err error) bool {
	merr, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}
	return merr.Number == errMySQLDuplicatedRecord
}

func isMySQLForeignKeyConstraintFailsError(err error) bool {
	merr, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}
	return merr.Number == errMySQLForeignKeyConstraintFails
}

func dbExists(tx *gorm.DB, where interface{}, tableName ...string) (exists bool, err error) {
	c := 0
	if len(tableName) > 0 {
		tx = tx.Table(tableName[0])
	} else {
		tx = tx.Model(where)
	}
	err = tx.Where(where).Limit(1).Count(&c).Error
	return c > 0, err
}

func convertError(err error) error {
	switch {
	case gorm.IsRecordNotFoundError(err):
		return ErrNotFound
	default:
		return err
	}
}
