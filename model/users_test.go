package model

import (
	"encoding/hex"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/utils"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users", (&User{}).TableName())
}

// TestParallelGroup6 並列テストグループ6 競合がないようなサブテストにすること
func TestParallelGroup6(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	// CreateUser
	t.Run("TestCreateUser", func(t *testing.T) {
		t.Parallel()

		_, err := CreateUser("あああ", "test@test.test", "test", role.User)
		assert.Error(err)
		_, err = CreateUser("aaa", "アイウエオ", "test", role.User)
		assert.Error(err)

		s := utils.RandAlphabetAndNumberString(10)
		user, err := CreateUser(s, "test@test.test", "test", role.User)
		if assert.NoError(err) {
			assert.NotEmpty(user.ID)
			assert.Equal(s, user.Name)
			assert.NotEmpty(user.Salt)
			assert.NotEmpty(user.Password)
			assert.Equal("test@test.test", user.Email)
			assert.Equal(role.User.ID(), user.Role)
		}

		_, err = CreateUser(s, "test@test.test", "test", role.User)
		assert.Error(err)
	})

	// GetUser
	t.Run("TestGetUser", func(t *testing.T) {
		t.Parallel()

		_, err := GetUser(uuid.Nil)
		assert.Error(err)

		u, err := GetUser(user.ID)
		if assert.NoError(err) {
			assert.Equal(user.ID, u.ID)
			assert.Equal(user.Name, u.Name)
		}
	})

	// GetUserByName
	t.Run("TestGetUserByName", func(t *testing.T) {
		t.Parallel()

		_, err := GetUserByName("")
		assert.Error(err)

		u, err := GetUserByName(user.Name)
		if assert.NoError(err) {
			assert.Equal(user.ID, u.ID)
			assert.Equal(user.Name, u.Name)
		}
	})

	// ChangeUserPassword
	t.Run("TestChangeUserPassword", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		newPass := "aiueo123"

		if assert.NoError(ChangeUserPassword(user.ID, newPass)) {
			u, err := GetUser(user.ID)
			require.NoError(err)

			salt, err := hex.DecodeString(u.Salt)
			require.NoError(err)
			assert.Equal(u.Password, hex.EncodeToString(hashPassword(newPass, salt)))
		}
	})

	// ChangeUserIcon
	t.Run("TestChangeUserIcon", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		newIcon := uuid.NewV4()
		if assert.NoError(ChangeUserIcon(user.ID, newIcon)) {
			u, err := GetUser(user.ID)
			require.NoError(err)
			assert.Equal(newIcon, u.Icon)
		}
	})

	// ChangeUserDisplayName
	t.Run("TestChangeUserDisplayName", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		newDN := uuid.NewV4().String()

		if assert.NoError(ChangeUserDisplayName(user.ID, newDN)) {
			u, err := GetUser(user.ID)
			require.NoError(err)
			assert.Equal(newDN, u.DisplayName)
		}

		assert.Error(ChangeUserDisplayName(user.ID, strings.Repeat("a", 100)))
	})

	// ChangeUserTwitterID
	t.Run("TestChangeUserTwitterID", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		newTwitter := "aiueo"

		if assert.NoError(ChangeUserTwitterID(user.ID, newTwitter)) {
			u, err := GetUser(user.ID)
			require.NoError(err)
			assert.Equal(newTwitter, u.TwitterID)
		}

		assert.Error(ChangeUserTwitterID(user.ID, "あああああ"))
	})

	// AuthenticateUser
	t.Run("TestAuthenticateUser", func(t *testing.T) {
		t.Parallel()

		u := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))

		assert.NoError(AuthenticateUser(u, "test"))
		assert.Error(AuthenticateUser(u, "wrong"))
		assert.Error(AuthenticateUser(nil, ""))
		u.Bot = true
		assert.Error(AuthenticateUser(u, "test"))
	})
}

func TestGetUsers(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	for i := 0; i < 5; i++ {
		mustMakeUser(t, "testGetUsers"+strconv.Itoa(i))
	}
	users, err := GetUsers()
	if assert.NoError(err) {
		// traqユーザー・テストユーザーがいるので
		assert.Len(users, 5+2)
	}
}
