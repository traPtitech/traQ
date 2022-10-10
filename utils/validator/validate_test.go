package validator

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/traPtitech/traQ/utils/optional"
)

func TestNotNilUUID(t *testing.T) {
	t.Parallel()

	t.Run("ok (nil)", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, NotNilUUID.Validate(nil))
	})
	t.Run("ok (uuid.UUID)", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, NotNilUUID.Validate(uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))))
	})
	t.Run("ok (uuid.NullUUID)", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, NotNilUUID.Validate(optional.From(uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000")))))
	})
	t.Run("ok (uuid.NullUUID Valid:false)", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, NotNilUUID.Validate(optional.Of[uuid.UUID]{}))
	})
	t.Run("ok (string)", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, NotNilUUID.Validate("550e8400-e29b-41d4-a716-446655440000"))
	})
	t.Run("ok ([]byte)", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, NotNilUUID.Validate(uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000")).Bytes()))
	})
	t.Run("ng (int)", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, NotNilUUID.Validate(1))
	})
	t.Run("ng (uuid.UUID)", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, NotNilUUID.Validate(uuid.Nil))
	})
	t.Run("ng (uuid.UUID)", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, NotNilUUID.Validate(optional.From(uuid.Nil)))
	})
	t.Run("ng (string)", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, NotNilUUID.Validate(uuid.Nil.String()))
	})
	t.Run("ng ([]byte)", func(t *testing.T) {
		t.Parallel()
		assert.Error(t, NotNilUUID.Validate(uuid.Nil.Bytes()))
	})
}
