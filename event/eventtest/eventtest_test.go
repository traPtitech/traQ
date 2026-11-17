package eventtest

import (
	"testing"

	"github.com/gofrs/uuid"
)

type validPayload struct {
	ID uuid.UUID `hub:"id"`
}

func (validPayload) TopicName() string { return "test.valid" }

// TestAssertValidPayload 規約を満たすペイロード型を受け付けることを確かめる
func TestAssertValidPayload(t *testing.T) {
	t.Parallel()
	AssertValidPayload(t, validPayload{ID: uuid.Must(uuid.NewV7())})
}
