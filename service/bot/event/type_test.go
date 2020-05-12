package event

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestType_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "event", Type("event").String())
}

func TestTypes_Value(t *testing.T) {
	t.Parallel()
	es := Types{"PING": struct{}{}, "PONG": struct{}{}}
	v, err := es.Value()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"PING", "PONG"}, strings.Split(v.(string), " "))
}

func TestTypes_Scan(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		s := Types{}
		assert.NoError(t, s.Scan(nil))
		assert.EqualValues(t, Types{}, s)
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		s := Types{}
		assert.NoError(t, s.Scan("a b c c  "))
		assert.Contains(t, s, Type("a"))
		assert.Contains(t, s, Type("b"))
		assert.Contains(t, s, Type("c"))
	})

	t.Run("[]byte", func(t *testing.T) {
		t.Parallel()

		s := Types{}
		assert.NoError(t, s.Scan([]byte("a b c c  ")))
		assert.Contains(t, s, Type("a"))
		assert.Contains(t, s, Type("b"))
		assert.Contains(t, s, Type("c"))
	})

	t.Run("other", func(t *testing.T) {
		t.Parallel()

		s := Types{}
		assert.Error(t, s.Scan(123))
	})
}

func TestTypes_String(t *testing.T) {
	t.Parallel()
	es := Types{"PING": struct{}{}, "PONG": struct{}{}}
	assert.ElementsMatch(t, []string{"PING", "PONG"}, strings.Split(es.String(), " "))
}

func TestTypes_Contains(t *testing.T) {
	t.Parallel()
	es := Types{"PING": struct{}{}, "PONG": struct{}{}}
	assert.True(t, es.Contains("PING"))
	assert.False(t, es.Contains("PAN"))
}

func TestTypes_MarshalJSON(t *testing.T) {
	t.Parallel()
	es := Types{"PING": struct{}{}, "PONG": struct{}{}}
	b, err := es.MarshalJSON()
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{`"PING"`, `"PONG"`}, strings.Split(strings.Trim(string(b), "[]"), ","))
}
