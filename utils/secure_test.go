// revive:disable-next-line FIXME: https://github.com/traPtitech/traQ/issues/2717
package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/traPtitech/traQ/utils/random"
)

func TestHashPassword(t *testing.T) {
	t.Parallel()

	password1 := "test"
	password2 := "testtest"
	salt1 := random.Salt()
	salt2 := random.Salt()

	assert.EqualValues(t, HashPassword(password1, salt1), HashPassword(password1, salt1))
	assert.NotEqual(t, HashPassword(password1, salt1), HashPassword(password1, salt2))
	assert.NotEqual(t, HashPassword(password2, salt1), HashPassword(password1, salt1))
}
