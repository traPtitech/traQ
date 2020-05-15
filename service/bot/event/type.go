package event

import (
	"database/sql/driver"
	"errors"
	vd "github.com/go-ozzo/ozzo-validation/v4"
	jsoniter "github.com/json-iterator/go"
	"strings"
)

// Type Botイベントタイプ
type Type string

func (t Type) String() string {
	return string(t)
}

func (t Type) Valid() bool {
	return allTypes.Contains(t)
}

// Types BotイベントタイプのSet
type Types map[Type]struct{}

func TypesFromArray(arr []string) Types {
	res := Types{}
	for _, v := range arr {
		if len(v) > 0 {
			res[Type(v)] = struct{}{}
		}
	}
	return res
}

// String event.Typesをスペース区切りで文字列に出力します
func (set Types) String() string {
	sa := make([]string, 0, len(set))
	for k := range set {
		sa = append(sa, string(k))
	}
	return strings.Join(sa, " ")
}

// Contains 指定したevent.Typeが含まれているかどうか
func (set Types) Contains(ev Type) bool {
	_, ok := set[ev]
	return ok
}

// Array event.Typesをstringの配列に変換します
func (set Types) Array() (r []string) {
	r = make([]string, 0, len(set))
	for s := range set {
		r = append(r, s.String())
	}
	return r
}

// Clone event.Typesを複製します
func (set Types) Clone() Types {
	dst := make(Types, len(set))
	for k, v := range set {
		dst[k] = v
	}
	return dst
}

// MarshalJSON encoding/json.Marshaler 実装
func (set Types) MarshalJSON() ([]byte, error) {
	return jsoniter.ConfigFastest.Marshal(set.Array())
}

// UnmarshalJSON encoding/json.Unmarshaler 実装
func (set *Types) UnmarshalJSON(data []byte) error {
	var arr []string
	err := jsoniter.ConfigFastest.Unmarshal(data, &arr)
	if err != nil {
		return err
	}
	*set = TypesFromArray(arr)
	return nil
}

// Value database/sql/driver.Valuer 実装
func (set Types) Value() (driver.Value, error) {
	return set.String(), nil
}

// Scan database/sql.Scanner 実装
func (set *Types) Scan(src interface{}) error {
	switch s := src.(type) {
	case nil:
		*set = Types{}
	case string:
		*set = TypesFromArray(strings.Split(s, " "))
	case []byte:
		*set = TypesFromArray(strings.Split(string(s), " "))
	default:
		return errors.New("failed to scan BotEvents")
	}
	return nil
}

// Validate ozzo-validation.Validatable 実装
func (set Types) Validate() error {
	if set == nil {
		return nil
	}
	return vd.Validate(set.Array(), vd.Each(vd.Required, vd.By(func(v interface{}) error {
		if !Type(v.(string)).Valid() {
			return errors.New("must be bot event")
		}
		return nil
	})))
}
