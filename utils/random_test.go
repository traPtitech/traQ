package utils

import "testing"

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
