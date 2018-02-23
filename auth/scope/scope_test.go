package scope

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValid(t *testing.T) {
	assert.True(t, Valid(Read))
	assert.True(t, Valid(Write))
	assert.False(t, Valid("fjeaowijfiow"))
}

func TestAccessScopes_Contains(t *testing.T) {
	s := AccessScopes{}
	s = append(s, Read, Write)

	assert.True(t, s.Contains(Read))
	assert.True(t, s.Contains(Write))
	assert.False(t, s.Contains(PrivateRead))
}

func TestAccessScopes_String(t *testing.T) {
	s := AccessScopes{}
	s = append(s, Read, Write)

	assert.EqualValues(t, "read write", s.String())
	assert.EqualValues(t, "", AccessScopes{}.String())
}
