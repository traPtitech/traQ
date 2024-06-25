package model

import (
	"encoding/hex"
	"testing"
	"testing/quick"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/random"

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

		tt := []UserInfo{
			&User{Bot: true},
			&User{},
			&User{Password: hex.EncodeToString(uuid.Must(uuid.NewV7()).Bytes()), Salt: "アイウエオ"},
			&User{Salt: hex.EncodeToString(uuid.Must(uuid.NewV7()).Bytes()), Password: "アイウエオ"},
			&User{Salt: hex.EncodeToString(uuid.Must(uuid.NewV7()).Bytes()), Password: hex.EncodeToString(uuid.Must(uuid.NewV7()).Bytes())},
		}
		for _, u := range tt {
			assert.Error(u.Authenticate("test"))
		}
	})

	t.Run("successes", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		tester := func(pass string) bool {
			salt := random.Salt()
			hashed := utils.HashPassword(pass, salt)
			u := &User{Password: hex.EncodeToString(hashed), Salt: hex.EncodeToString(salt)}
			return u.Authenticate(pass) == nil
		}

		assert.NoError(quick.Check(tester, &quick.Config{MaxCount: 10}))
	})
}

func TestUserAccountStatus_Valid(t *testing.T) {
	t.Parallel()

	assert.True(t, UserAccountStatusDeactivated.Valid())
	assert.False(t, UserAccountStatus(-1).Valid())
}
