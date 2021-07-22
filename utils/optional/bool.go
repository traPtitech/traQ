package optional

import (
	"bytes"
	"database/sql"
	"errors"

	jsoniter "github.com/json-iterator/go"
)

type Bool struct {
	sql.NullBool
}

func BoolFrom(v bool) Bool {
	return NewBool(v, true)
}

func NewBool(v bool, valid bool) Bool {
	return Bool{NullBool: sql.NullBool{Bool: v, Valid: valid}}
}

func (b Bool) ValueOrZero() bool {
	if b.Valid {
		return b.Bool
	}
	return false
}

func (b *Bool) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		b.Bool, b.Valid = false, false
		return nil
	}

	if err := jsoniter.ConfigFastest.Unmarshal(data, &b.Bool); err != nil {
		return err
	}

	b.Valid = true
	return nil
}

func (b Bool) MarshalJSON() ([]byte, error) {
	if b.Valid {
		if b.Bool {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	}
	return []byte("null"), nil
}

func (b *Bool) UnmarshalText(text []byte) error {
	str := string(text)
	switch str {
	case "", "null":
		b.Bool = false
		b.Valid = false
		return nil
	case "true":
		b.Bool = true
		b.Valid = true
		return nil
	case "false":
		b.Bool = false
		b.Valid = true
		return nil
	default:
		b.Bool = false
		b.Valid = false
		return errors.New("invalid input:" + str)
	}
}

func (b Bool) MarshalText() ([]byte, error) {
	if b.Valid {
		if b.Bool {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	}
	return []byte{}, nil
}
