package webrtcv3

import (
	"testing"

	"github.com/traPtitech/traQ/event/eventtest"
)

// TestPayloads この package で定義しているペイロード型が規約を満たしているかを確かめる。
// ペイロード型を追加したらここにも追加すること。
func TestPayloads(t *testing.T) {
	t.Parallel()
	eventtest.AssertValidPayload(t, UserWebRTCv3StateChangedPayload{})
}
