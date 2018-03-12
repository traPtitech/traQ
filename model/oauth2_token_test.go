package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOAuth2Token_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "oauth2_tokens", (&OAuth2Token{}).TableName())
}
