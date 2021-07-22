package set

import (
	"strings"

	"github.com/gofrs/uuid"
)

// UUID uuid.UUIDの集合
type UUID map[uuid.UUID]struct{}

// Add 要素を追加します
func (set UUID) Add(v ...uuid.UUID) {
	for _, v := range v {
		set[v] = struct{}{}
	}
}

// Remove 要素を削除します
func (set UUID) Remove(v ...uuid.UUID) {
	for _, v := range v {
		delete(set, v)
	}
}

// String 要素をsep区切りで文字列に出力します
func (set UUID) String(sep string) string {
	sa := make([]string, 0, len(set))
	for k := range set {
		sa = append(sa, k.String())
	}
	return strings.Join(sa, sep)
}

// Contains 指定した要素が含まれているかどうか
func (set UUID) Contains(v uuid.UUID) bool {
	_, ok := set[v]
	return ok
}

// MarshalJSON encoding/json.Marshaler 実装
func (set UUID) MarshalJSON() ([]byte, error) {
	arr := make([]uuid.UUID, 0, len(set))
	for e := range set {
		arr = append(arr, e)
	}
	return json.Marshal(arr)
}

// UnmarshalJSON encoding/json.Unmarshaler 実装
func (set *UUID) UnmarshalJSON(data []byte) error {
	var value []uuid.UUID
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*set = UUIDSetFromArray(value)
	return nil
}

// Clone 集合を複製します
func (set UUID) Clone() UUID {
	a := UUID{}
	for k, v := range set {
		a[k] = v
	}
	return a
}

// StringArray stringのスライスに変換します
func (set UUID) StringArray() []string {
	arr := make([]string, 0, len(set))
	for k := range set {
		arr = append(arr, k.String())
	}
	return arr
}

// Array uuid.UUIDのスライスに変換します
func (set UUID) Array() []uuid.UUID {
	arr := make([]uuid.UUID, 0, len(set))
	for k := range set {
		arr = append(arr, k)
	}
	return arr
}

// Plus 集合を足します
func (set UUID) Plus(sets ...UUID) {
	for _, s := range sets {
		for k := range s {
			set[k] = struct{}{}
		}
	}
}

// UnionUUIDSets 集合の和集合を返します
func UnionUUIDSets(sets ...UUID) UUID {
	result := UUID{}
	for _, s := range sets {
		for k := range s {
			result[k] = struct{}{}
		}
	}
	return result
}

// UUIDSetFromArray 配列から集合を生成します
func UUIDSetFromArray(arr []uuid.UUID) UUID {
	s := UUID{}
	s.Add(arr...)
	return s
}
