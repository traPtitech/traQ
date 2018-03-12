package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOAuth2Client_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "oauth2_clients", (&OAuth2Client{}).TableName())
}
