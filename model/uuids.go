package model

import (
	"database/sql/driver"
	"errors"
	"strings"

	"github.com/gofrs/uuid"
)

type UUIDs []uuid.UUID

func (arr UUIDs) Value() (driver.Value, error) {
	idStr := make([]string, len(arr))
	for i, id := range arr {
		idStr[i] = id.String()
	}
	return strings.Join(idStr, ","), nil
}

func (arr *UUIDs) Scan(src interface{}) error {
	switch s := src.(type) {
	case nil:
		*arr = UUIDs{}
	case string:
		for _, value := range strings.Split(s, ",") {
			ID, err := uuid.FromString(value)
			if err != nil {
				continue
			}
			*arr = append(*arr, ID)
		}
	case []byte:
		for _, value := range strings.Split(string(s), ",") {
			ID, err := uuid.FromString(value)
			if err != nil {
				continue
			}
			*arr = append(*arr, ID)
		}
	default:
		return errors.New("failed to scan UUIDs")
	}
	return nil
}

func (arr UUIDs) ToUUIDSlice() []uuid.UUID {
	return arr
}
