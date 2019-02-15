package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHashPassword(t *testing.T) {
	t.Parallel()

	password1 := "test"
	password2 := "testtest"
	salt1 := GenerateSalt()
	salt2 := GenerateSalt()

	assert.EqualValues(t, HashPassword(password1, salt1), HashPassword(password1, salt1))
	assert.NotEqual(t, HashPassword(password1, salt1), HashPassword(password1, salt2))
	assert.NotEqual(t, HashPassword(password2, salt1), HashPassword(password1, salt1))
}
