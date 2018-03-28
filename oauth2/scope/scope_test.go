package scope

import (
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/rbac/role"
	"regexp"
	"testing"
)

func TestScopes(t *testing.T) {
	t.Parallel()

	regex := regexp.MustCompile(`^[\x21\x23-\x5B\x5D-\x7E]+$`)
	for k := range list {
		assert.True(t, regex.MatchString(string(k)))
	}
}

func TestValid(t *testing.T) {
	t.Parallel()

	assert.True(t, Valid(Read))
	assert.True(t, Valid(Write))
	assert.False(t, Valid("fjeaowijfiow"))
}

func TestAccessScopes_Contains(t *testing.T) {
	t.Parallel()

	s := AccessScopes{}
	s = append(s, Read, Write)

	assert.True(t, s.Contains(Read))
	assert.True(t, s.Contains(Write))
	assert.False(t, s.Contains(PrivateRead))
}

func TestAccessScopes_String(t *testing.T) {
	t.Parallel()

	s := AccessScopes{}
	s = append(s, Read, Write)

	assert.EqualValues(t, "read write", s.String())
	assert.EqualValues(t, "", AccessScopes{}.String())
}

func TestAccessScopes_GenerateRole(t *testing.T) {
	t.Parallel()

	s := AccessScopes{}
	s = append(s, Read, Write)
	assert.Contains(t, s.GenerateRole().ID(), role.ReadUser.ID())
	assert.Contains(t, s.GenerateRole().ID(), role.WriteUser.ID())
}
