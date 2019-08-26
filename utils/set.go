package utils

import (
	"encoding/json"
	"strings"
)

// StringSet stringの集合
type StringSet map[string]struct{}

// StringSetFromArray 配列から集合を生成します
func StringSetFromArray(arr []string) StringSet {
	s := StringSet{}
	for _, v := range arr {
		s.Add(v)
	}
	return s
}

// Add 要素を追加します
func (set StringSet) Add(str string) {
	set[str] = struct{}{}
}

// Remove 要素を削除します
func (set StringSet) Remove(str string) {
	delete(set, str)
}

// String 要素をsep区切りで文字列に出力します
func (set StringSet) String(sep string) string {
	sa := make([]string, 0, len(set))
	for k := range set {
		sa = append(sa, string(k))
	}
	return strings.Join(sa, sep)
}

// Contains 指定した要素が含まれているかどうか
func (set StringSet) Contains(str string) bool {
	_, ok := set[str]
	return ok
}

// MarshalJSON encoding/json.Marshaler 実装
func (set StringSet) MarshalJSON() ([]byte, error) {
	arr := make([]string, 0, len(set))
	for e := range set {
		arr = append(arr, string(e))
	}
	return json.Marshal(arr)
}

// UnmarshalJSON encoding/json.Unmarshaler 実装
func (set *StringSet) UnmarshalJSON(data []byte) error {
	var value []string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*set = StringSetFromArray(value)
	return nil
}

// Clone 集合を複製します
func (set StringSet) Clone() StringSet {
	a := StringSet{}
	for k, v := range set {
		a[k] = v
	}
	return a
}
