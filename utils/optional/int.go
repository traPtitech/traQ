package optional

import (
	"bytes"
	"database/sql"
	"strconv"

	jsoniter "github.com/json-iterator/go"
)

type Int struct {
	sql.NullInt64
}

func IntFrom(v int64) Int {
	return NewInt(v, true)
}

func NewInt(v int64, valid bool) Int {
	return Int{NullInt64: sql.NullInt64{Int64: v, Valid: valid}}
}

func (i Int) ValueOrZero() int64 {
	if i.Valid {
		return i.Int64
	}
	return 0
}

func (i *Int) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		i.Int64, i.Valid = 0, false
		return nil
	}

	if err := jsoniter.ConfigFastest.Unmarshal(data, &i.Int64); err != nil {
		return err
	}

	i.Valid = true
	return nil
}

func (i Int) MarshalJSON() ([]byte, error) {
	if i.Valid {
		return jsoniter.ConfigFastest.Marshal(i.Int64)
	}
	return jsoniter.ConfigFastest.Marshal(nil)
}

func (i *Int) UnmarshalText(text []byte) error {
	str := string(text)
	if str == "" || str == "null" {
		i.Valid = false
		return nil
	}
	var err error
	i.Int64, err = strconv.ParseInt(string(text), 10, 64)
	i.Valid = err == nil
	return err
}

func (i Int) MarshalText() ([]byte, error) {
	if i.Valid {
		return []byte(strconv.FormatInt(i.Int64, 10)), nil
	}
	return []byte{}, nil
}
