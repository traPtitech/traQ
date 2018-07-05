package model

import (
	"encoding/hex"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/rbac/role"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users", (&User{}).TableName())
}

func TestCreateUser(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	_, err := CreateUser("あああ", "test@test.test", "test", role.User)
	assert.Error(err)
	_, err = CreateUser("aaa", "アイウエオ", "test", role.User)
	assert.Error(err)

	user, err := CreateUser("newUser", "test@test.test", "test", role.User)
	if assert.NoError(err) {
		assert.NotEmpty(user.ID)
		assert.Equal("newUser", user.Name)
		assert.NotEmpty(user.Salt)
		assert.NotEmpty(user.Password)
		assert.Equal("test@test.test", user.Email)
		assert.Equal(role.User.ID(), user.Role)
	}

	_, err = CreateUser("newUser", "test@test.test", "test", role.User)
	assert.Error(err)
}

func TestGetUser(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	_, err := GetUser(uuid.Nil)
	assert.Error(err)

	u, err := GetUser(user.GetUID())
	if assert.NoError(err) {
		assert.Equal(user.ID, u.ID)
		assert.Equal(user.Name, u.Name)
	}
}

func TestGetUserByName(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	_, err := GetUserByName("")
	assert.Error(err)

	u, err := GetUserByName(user.Name)
	if assert.NoError(err) {
		assert.Equal(user.ID, u.ID)
		assert.Equal(user.Name, u.Name)
	}
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

func TestChangeUserPassword(t *testing.T) {
	assert, require, user, _ := beforeTest(t)
	newPass := "aiueo123"

	if assert.NoError(ChangeUserPassword(user.GetUID(), newPass)) {
		u, err := GetUser(user.GetUID())
		require.NoError(err)

		salt, err := hex.DecodeString(u.Salt)
		require.NoError(err)
		assert.Equal(u.Password, hex.EncodeToString(hashPassword(newPass, salt)))
	}
}

func TestChangeUserIcon(t *testing.T) {
	assert, require, user, _ := beforeTest(t)
	newIcon := uuid.NewV4()

	if assert.NoError(ChangeUserIcon(user.GetUID(), newIcon)) {
		u, err := GetUser(user.GetUID())
		require.NoError(err)
		assert.Equal(newIcon.String(), u.Icon)
	}
}

func TestChangeUserDisplayName(t *testing.T) {
	assert, require, user, _ := beforeTest(t)
	newDN := uuid.NewV4().String()

	if assert.NoError(ChangeUserDisplayName(user.GetUID(), newDN)) {
		u, err := GetUser(user.GetUID())
		require.NoError(err)
		assert.Equal(newDN, u.DisplayName)
	}

	assert.Error(ChangeUserDisplayName(user.GetUID(), strings.Repeat("a", 100)))
}

func TestChangeUserTwitterID(t *testing.T) {
	assert, require, user, _ := beforeTest(t)
	newTwitter := "aiueo"

	if assert.NoError(ChangeUserTwitterID(user.GetUID(), newTwitter)) {
		u, err := GetUser(user.GetUID())
		require.NoError(err)
		assert.Equal(newTwitter, u.TwitterID)
	}

	assert.Error(ChangeUserTwitterID(user.GetUID(), "あああああ"))
}

func TestAuthenticateUser(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	u := mustMakeUser(t, "testAuthorization")

	assert.NoError(AuthenticateUser(u, "test"))
	assert.Error(AuthenticateUser(u, "wrong"))
	assert.Error(AuthenticateUser(nil, ""))
	u.Bot = true
	assert.Error(AuthenticateUser(u, "test"))
}
