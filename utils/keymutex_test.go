// revive:disable-next-line FIXME: https://github.com/traPtitech/traQ/issues/2717
package utils

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewKeyMutex(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	km := NewKeyMutex(10)
	if assert.NotNil(km) {
		assert.EqualValues(10, km.count)
		assert.Len(km.locks, 10)
	}
}

func TestKeyMutex_LockAndUnlock(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	km := NewKeyMutex(10)

	counter := [10]int{}
	keys := []string{
		"test",
		"aiueo",
		"abcd",
		"12345",
		"foo",
		"bar",
		"a",
		"b",
		"1111",
		"eeee",
	}

	wg := sync.WaitGroup{}
	for i := range 100000 {
		wg.Add(1)
		go func(i int) {
			j := i % 10
			km.Lock(keys[j])
			counter[j]++
			km.Unlock(keys[j])
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := range 10 {
		assert.Equal(10000, counter[i])
	}
}
