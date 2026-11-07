package model

import (
	"database/sql/driver"
	"errors"

	"github.com/traPtitech/traQ/utils/jsonutil"
)

type JSON map[string]interface{}

// Value database/sql/driver.Valuer 実装
func (v JSON) Value() (driver.Value, error) {
	return jsonutil.MarshalToString(v)
}

// Scan database/sql.Scanner 実装
func (v *JSON) Scan(src interface{}) error {
	switch s := src.(type) {
	case nil:
		return nil
	case string:
		return jsonutil.Unmarshal([]byte(s), v)
	case []byte:
		return jsonutil.Unmarshal(s, v)
	default:
		return errors.New("failed to scan JSON")
	}
}
