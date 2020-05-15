package set

import (
	"strings"
)

// String stringの集合
type String map[string]struct{}

// Add 要素を追加します
func (set String) Add(v ...string) {
	for _, v := range v {
		set[v] = struct{}{}
	}
}

// Remove 要素を削除します
func (set String) Remove(v ...string) {
	for _, v := range v {
		delete(set, v)
	}
}

// String 要素をsep区切りで文字列に出力します
func (set String) String(sep string) string {
	sa := make([]string, 0, len(set))
	for k := range set {
		sa = append(sa, k)
	}
	return strings.Join(sa, sep)
}

// Contains 指定した要素が含まれているかどうか
func (set String) Contains(v string) bool {
	_, ok := set[v]
	return ok
}

// MarshalJSON encoding/json.Marshaler 実装
func (set String) MarshalJSON() ([]byte, error) {
	arr := make([]string, 0, len(set))
	for e := range set {
		arr = append(arr, e)
	}
	return json.Marshal(arr)
}

// UnmarshalJSON encoding/json.Unmarshaler 実装
func (set *String) UnmarshalJSON(data []byte) error {
	var value []string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*set = StringSetFromArray(value)
	return nil
}

// Clone 集合を複製します
func (set String) Clone() String {
	a := String{}
	for k, v := range set {
		a[k] = v
	}
	return a
}

// StringSetFromArray 配列から集合を生成します
func StringSetFromArray(arr []string) String {
	s := String{}
	s.Add(arr...)
	return s
}
