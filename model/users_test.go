package model

import (
	"encoding/hex"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
)

func TestUser_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users", (&User{}).TableName())
}

func TestAuthenticateUser(t *testing.T) {
	t.Parallel()

	t.Run("failures", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.Error(AuthenticateUser(nil, "test"))
		assert.Error(AuthenticateUser(&User{Bot: true}, "test"))
		assert.Error(AuthenticateUser(&User{}, "test"))
		assert.Error(AuthenticateUser(&User{Password: hex.EncodeToString(uuid.NewV4().Bytes()), Salt: "アイウエオ"}, "test"))
		assert.Error(AuthenticateUser(&User{Salt: hex.EncodeToString(uuid.NewV4().Bytes()), Password: "アイウエオ"}, "test"))
		assert.Error(AuthenticateUser(&User{Salt: hex.EncodeToString(uuid.NewV4().Bytes()), Password: hex.EncodeToString(uuid.NewV4().Bytes())}, "test"))
	})

	t.Run("successes", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		tester := func(pass string) bool {
			salt := utils.GenerateSalt()
			hashed := utils.HashPassword(pass, salt)
			return AuthenticateUser(&User{Password: hex.EncodeToString(hashed), Salt: hex.EncodeToString(salt)}, pass) == nil
		}

		assert.NoError(quick.Check(tester, &quick.Config{MaxCount: 10}))
	})
}
