// Package eventtest event.Payload を実装する型をテストするためのヘルパー
package eventtest

import (
	"fmt"
	"testing"

	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/event"
)

// AssertValidPayload ペイロード型 T の定義が規約を満たし、
// event.FieldsOf で変換した値を event.PayloadOf で元の値に戻せることを確かめる。
// event package の外で定義したペイロード型は、定義した package のテストからこれを呼ぶこと。
// interface 型のフィールドはゼロ値 (nil) だとデコードできないため、値を入れて渡すこと。
func AssertValidPayload[T event.Payload](t *testing.T, p T) {
	t.Helper()
	t.Run(fmt.Sprintf("%T", p), func(t *testing.T) {
		var f hub.Fields
		require.NotPanics(t, func() { f = event.FieldsOf(p) })
		var (
			got T
			ok  bool
		)
		require.NotPanics(t, func() { got, ok = event.PayloadOf[T](f) })
		require.True(t, ok)
		assert.Equal(t, p, got)
	})
}
