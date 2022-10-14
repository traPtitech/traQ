package optional

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/guregu/null"
	jsonIter "github.com/json-iterator/go"
)

// Of nullableなjsonフィールドとして使用できます。
// json.Unmarshaler, json.Marshaler, encoding.TextUnmarshaler, encoding.TextMarshaler, sql.Scanner, driver.Valuer
// を実装する型と、一部の型についてこれらのメソッドを使用できます。
type Of[T any] struct {
	V     T
	Valid bool
}

func New[T any](v T, valid bool) Of[T] {
	return Of[T]{
		V:     v,
		Valid: valid,
	}
}

func From[T any](v T) Of[T] {
	return Of[T]{
		V:     v,
		Valid: true,
	}
}

// ValueOrZero 値が入っているときはその値を、そうでないときはゼロ値を返します。
func (o Of[T]) ValueOrZero() T {
	if o.Valid {
		return o.V
	}
	var t T
	return t
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (o *Of[T]) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		var t T
		o.V, o.Valid = t, false
		return nil
	}

	if u, ok := any(&o.V).(json.Unmarshaler); ok {
		if err := u.UnmarshalJSON(data); err != nil {
			return err
		}
		o.Valid = true
		return nil
	}
	if err := jsonIter.ConfigFastest.Unmarshal(data, &o.V); err != nil {
		return err
	}
	o.Valid = true
	return nil
}

// MarshalJSON implements json.Marshaler interface.
func (o Of[T]) MarshalJSON() ([]byte, error) {
	if !o.Valid {
		return jsonIter.ConfigFastest.Marshal(nil)
	}
	if m, ok := any(o.V).(json.Marshaler); ok {
		return m.MarshalJSON()
	}
	return jsonIter.ConfigFastest.Marshal(o.V)
}

// UnmarshalText implements encoding.TextUnmarshaler interface.
func (o *Of[T]) UnmarshalText(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		var t T
		o.V = t
		o.Valid = false
		return nil
	}

	switch any(o.V).(type) {
	case bool:
		var b null.Bool
		if err := b.UnmarshalText(data); err != nil {
			return err
		}
		o.V, o.Valid = any(b.Bool).(T), true
		return nil
	case int:
		var i null.Int
		if err := i.UnmarshalText(data); err != nil {
			return err
		}
		o.V, o.Valid = any(int(i.Int64)).(T), true
		return nil
	case string:
		var s null.String
		if err := s.UnmarshalText(data); err != nil {
			return err
		}
		o.V, o.Valid = any(s.String).(T), true
		return nil
	default:
		t, ok := any(&o.V).(encoding.TextUnmarshaler)
		if !ok {
			return fmt.Errorf("unsupported type for UnmarshalText: %T", t)
		}
		if err := t.UnmarshalText(data); err != nil {
			return err
		}
		o.Valid = true
		return nil
	}
}

// MarshalText implements encoding.Marshaler interface.
func (o Of[T]) MarshalText() ([]byte, error) {
	if !o.Valid {
		return []byte{}, nil
	}

	switch v := any(o.V).(type) {
	case bool:
		if v {
			return []byte("true"), nil
		}
		return []byte("false"), nil
	case int:
		return []byte(strconv.FormatInt(int64(v), 10)), nil
	case string:
		return []byte(v), nil
	default:
		t, ok := v.(encoding.TextMarshaler)
		if !ok {
			return nil, fmt.Errorf("unsupported type for MarshalText: %T", t)
		}
		return t.MarshalText()
	}
}

// Scan implements sql.Scanner interface.
func (o *Of[T]) Scan(src any) error {
	switch any(o.V).(type) {
	case bool:
		var b sql.NullBool
		if err := b.Scan(src); err != nil {
			return err
		}
		o.V, o.Valid = any(b.Bool).(T), b.Valid
		return nil
	case int:
		var i sql.NullInt64
		if err := i.Scan(src); err != nil {
			return err
		}
		o.V, o.Valid = any(int(i.Int64)).(T), i.Valid
		return nil
	case string:
		var s sql.NullString
		if err := s.Scan(src); err != nil {
			return err
		}
		o.V, o.Valid = any(s.String).(T), s.Valid
		return nil
	case time.Time:
		var t sql.NullTime
		if err := t.Scan(src); err != nil {
			return err
		}
		o.V, o.Valid = any(t.Time).(T), t.Valid
		return nil
	default:
		s, ok := any(&o.V).(sql.Scanner)
		if !ok {
			return fmt.Errorf("unsupported type for Scan: %T", o.V)
		}
		if err := s.Scan(src); err != nil {
			return err
		}
		o.Valid = true
		return nil
	}
}

// Value implements driver.Valuer interface.
func (o Of[T]) Value() (driver.Value, error) {
	if !o.Valid {
		return nil, nil
	}
	switch v := any(o.V).(type) {
	case int:
		return int64(v), nil
	case driver.Valuer:
		return v.Value()
	}
	return o.V, nil
}
