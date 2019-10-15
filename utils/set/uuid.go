package set

import (
	"github.com/gofrs/uuid"
	"strings"
)

// UUIDSet uuid.UUIDの集合
type UUIDSet map[uuid.UUID]struct{}

// UUIDSetFromArray 配列から集合を生成します
func UUIDSetFromArray(arr []uuid.UUID) UUIDSet {
	s := UUIDSet{}
	s.Add(arr...)
	return s
}

// Add 要素を追加します
func (set UUIDSet) Add(v ...uuid.UUID) {
	for _, v := range v {
		set[v] = struct{}{}
	}
}

// Remove 要素を削除します
func (set UUIDSet) Remove(v ...uuid.UUID) {
	for _, v := range v {
		delete(set, v)
	}
}

// String 要素をsep区切りで文字列に出力します
func (set UUIDSet) String(sep string) string {
	sa := make([]string, 0, len(set))
	for k := range set {
		sa = append(sa, k.String())
	}
	return strings.Join(sa, sep)
}

// Contains 指定した要素が含まれているかどうか
func (set UUIDSet) Contains(v uuid.UUID) bool {
	_, ok := set[v]
	return ok
}

// MarshalJSON encoding/json.Marshaler 実装
func (set UUIDSet) MarshalJSON() ([]byte, error) {
	arr := make([]uuid.UUID, 0, len(set))
	for e := range set {
		arr = append(arr, e)
	}
	return json.Marshal(arr)
}

// UnmarshalJSON encoding/json.Unmarshaler 実装
func (set *UUIDSet) UnmarshalJSON(data []byte) error {
	var value []uuid.UUID
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*set = UUIDSetFromArray(value)
	return nil
}

// Clone 集合を複製します
func (set UUIDSet) Clone() UUIDSet {
	a := UUIDSet{}
	for k, v := range set {
		a[k] = v
	}
	return a
}

// StringArray stringのスライスに変換します
func (set UUIDSet) StringArray() []string {
	arr := make([]string, 0, len(set))
	for k := range set {
		arr = append(arr, k.String())
	}
	return arr
}
