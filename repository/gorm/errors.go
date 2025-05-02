package gorm

import (
	"gorm.io/gorm"

	"github.com/traPtitech/traQ/repository"
)

func convertError(err error) error {
	switch err {
	case gorm.ErrRecordNotFound:
		return repository.ErrNotFound
	default:
		return err
	}
}
