package random

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandAlphabetAndNumberString(t *testing.T) {
	t.Parallel()

	set := make(map[string]bool, 1000)
	for range 1000 {
		s := AlphaNumeric(10)
		if set[s] {
			t.FailNow()
		}
		set[s] = true
	}
}

func TestSecureRandAlphabetAndNumberString(t *testing.T) {
	t.Parallel()

	set := make(map[string]bool, 1000)
	for range 1000 {
		s := SecureAlphaNumeric(10)
		if set[s] {
			t.FailNow()
		}
		set[s] = true
	}
}

func TestGenerateSalt(t *testing.T) {
	t.Parallel()

	salt := Salt()
	assert.Len(t, salt, 64)
	assert.NotEqual(t, salt, Salt())
}
