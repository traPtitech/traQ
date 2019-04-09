package utils

import (
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func TestNewKeyMutex(t *testing.T) {
	t.Parallel()
	assert := assert.New(t)

	km := NewKeyMutex(10)
	if assert.NotNil(km) {
		assert.Equal(10, km.count)
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
	for i := 0; i < 1000; i++ {
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

	for i := 0; i < 10; i++ {
		assert.Equal(100, counter[i])
	}
}
