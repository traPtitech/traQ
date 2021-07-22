package optional

import (
	"bytes"
	"database/sql"
	"time"

	jsoniter "github.com/json-iterator/go"
)

type Time struct {
	sql.NullTime
}

func TimeFrom(v time.Time) Time {
	return NewTime(v, true)
}

func NewTime(v time.Time, valid bool) Time {
	return Time{NullTime: sql.NullTime{Time: v, Valid: valid}}
}

func (t Time) ValueOrZero() time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Time{}
}

func (t *Time) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		t.Time, t.Valid = time.Time{}, false
		return nil
	}

	if err := jsoniter.ConfigFastest.Unmarshal(data, &t.Time); err != nil {
		return err
	}

	t.Valid = true
	return nil
}

func (t Time) MarshalJSON() ([]byte, error) {
	if t.Valid {
		return t.Time.MarshalJSON()
	}
	return jsoniter.ConfigFastest.Marshal(nil)
}

func (t *Time) UnmarshalText(text []byte) error {
	str := string(text)
	if str == "" || str == "null" {
		t.Valid = false
		return nil
	}
	if err := t.Time.UnmarshalText(text); err != nil {
		return err
	}
	t.Valid = true
	return nil
}

func (t Time) MarshalText() ([]byte, error) {
	if t.Valid {
		return t.Time.MarshalText()
	}
	return []byte{}, nil
}
