package model

import (
	"database/sql/driver"
	"errors"
)

type JSON map[string]interface{}

// Value database/sql/driver.Valuer 実装
func (v JSON) Value() (driver.Value, error) {
	return json.MarshalToString(v)
}

// Scan database/sql.Scanner 実装
func (v *JSON) Scan(src interface{}) error {
	switch s := src.(type) {
	case nil:
		return nil
	case string:
		return json.Unmarshal([]byte(s), v)
	case []byte:
		return json.Unmarshal(s, v)
	default:
		return errors.New("failed to scan JSON")
	}
}
