package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRandAlphabetAndNumberString(t *testing.T) {
	t.Parallel()

	set := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		s := RandAlphabetAndNumberString(10)
		if set[s] {
			t.FailNow()
		}
		set[s] = true
	}
}

func TestSecureRandAlphabetAndNumberString(t *testing.T) {
	t.Parallel()

	set := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		s := SecureRandAlphabetAndNumberString(10)
		if set[s] {
			t.FailNow()
		}
		set[s] = true
	}
}

func TestGenerateSalt(t *testing.T) {
	t.Parallel()

	salt := GenerateSalt()
	assert.Len(t, salt, 64)
	assert.NotEqual(t, salt, GenerateSalt())
}
