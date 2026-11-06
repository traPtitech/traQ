package etag

import (
	"crypto/md5"
	"encoding/hex"

	"github.com/traPtitech/traQ/utils/jsonutil"
)

func FromBytes(b []byte) string {
	md5Res := md5.Sum(b)

	return hex.EncodeToString(md5Res[:])
}

type Entity[T any] struct {
	value T
	etag  string
}

func NewEntity[T any](data T) (*Entity[T], error) {
	// 順番が一致しないとEtagが一致しないが、jsonutil.Marshalはマップのキーをソートする
	b, err := jsonutil.Marshal(data)
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
