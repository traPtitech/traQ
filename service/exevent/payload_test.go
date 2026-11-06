package exevent

import (
	"testing"

	"github.com/traPtitech/traQ/event/eventtest"
	"github.com/traPtitech/traQ/service/message"
)

// TestPayloads この package で定義しているペイロード型が規約を満たしているかを確かめる。
// ペイロード型を追加したらここにも追加すること。
func TestPayloads(t *testing.T) {
	t.Parallel()
	// interface 型のフィールドはゼロ値 (nil) だとデコードできないため、値を入れる
	eventtest.AssertValidPayload(t, MessageStampsUpdatedPayload{Message: fakeMessage{}})
}

type fakeMessage struct {
	message.Message
}
