package etag

import (
	"crypto/md5"
	"encoding/hex"

	jsonIter "github.com/json-iterator/go"
)

var JSONIter = jsonIter.Config{
	EscapeHTML:                    false,
	MarshalFloatWith6Digits:       true,
	ObjectFieldMustBeSimpleString: true,
	// ここより上はjsonIter.ConfigFastestと同様
	SortMapKeys: true, // 順番が一致しないとEtagが一致しないのでソートを有効にする
}.Froze()

func FromBytes(b []byte) string {
	md5Res := md5.Sum(b)

	return hex.EncodeToString(md5Res[:])
}

type Entity[T any] struct {
	value T
	etag  string
}

func NewEntity[T any](data T) (*Entity[T], error) {
	b, err := JSONIter.Marshal(data)
	if err != nil {
		return nil, err
	}

	md5Res := md5.Sum(b)

	return &Entity[T]{
		value: data,
		etag:  hex.EncodeToString(md5Res[:]),
	}, nil
}

func (e Entity[T]) Value() T {
	return e.value
}

func (e Entity[T]) ETag() string {
	return e.etag
}
