package model

import (
	"encoding/hex"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	password = "test"
)

func TestUser_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users", (&User{}).TableName())
}

func TestUser_Create(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	assert.Error((&User{}).Create())
	assert.Error((&User{Name: "test"}).Create())
	assert.Error((&User{Name: "test", Email: "test@test.test"}).Create())

	assert.Error((&User{Name: "test", Email: "test@test.test", Password: "test", Salt: "test"}).Create())
	user := &User{Name: "test", Email: "test@test.test", Password: "test", Salt: "test", Role: "user"}
	if assert.NoError(user.Create()) {
		assert.NotEmpty(user.ID)
	}
}

func TestUser_SetPassword(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	user := mustMakeUser(t, "testSetPassword")
	assert.NoError(checkEmptyField(user))

	salt, err := hex.DecodeString(user.Salt)
	assert.NoError(err)

	assert.Equal(user.Password, hex.EncodeToString(hashPassword(password, salt)))
}

func TestUser_Exists(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	user := mustMakeUser(t, "testExists")
	exists, err := user.Exists()
	assert.NoError(err)
	assert.True(exists)

	user = &User{Name: "no such user!"}
	exists, err = user.Exists()
	assert.NoError(err)
	assert.False(exists)

	user = &User{}
	exists, err = user.Exists()
	assert.Error(err)
}

func TestUser_GetUser(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	// 正常系
	user := mustMakeUser(t, "testGetUser")
	getUser, err := GetUser(user.ID)
	assert.NoError(err)

	// DB格納時に記録されるデータをコピー
	user.CreatedAt = getUser.CreatedAt
	user.UpdatedAt = getUser.UpdatedAt
	assert.EqualValues(user, getUser)

	// 異常系
	_, err = GetUser("wrong_id")
	assert.Error(err)
}

func TestUser_GetUsers(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	for i := 0; i < 5; i++ {
		mustMakeUser(t, "testGetUsers"+strconv.Itoa(i))
	}
	users, err := GetUsers()
	assert.NoError(err)

	// traqユーザー・テストユーザーがいるので
	assert.Len(users, 5+2)
}

func TestUser_Authorization(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	mustMakeUser(t, "testAuthorization")
	user := &User{Name: "testAuthorization"}

	assert.NoError(user.Authorization(password))
	assert.NoError(checkEmptyField(user))

	assert.Error(user.Authorization("invalid password"))

	assert.Error((&User{Name: "no such user!"}).Authorization(password))

	bot := mustMakeUser(t, "testAuthBot")
	bot.Bot = true

	assert.NoError(bot.Update())
	assert.Error(bot.Authorization(password))
}

func TestUser_Update(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	user := mustMakeUser(t, "testUpdate")
	invalidUser := &User{Name: "!nvalid"}

	icon, err := GenerateIcon(user.Name)
	assert.NoError(err)

	assert.NoError(user.UpdateIconID(icon))
	assert.Error(invalidUser.UpdateIconID(icon))

	name := "hogehoge"
	assert.NoError(user.UpdateDisplayName(name))
	assert.Error(invalidUser.UpdateDisplayName(name))

	user, err = GetUser(user.ID)
	assert.Equal(user.DisplayName, name)
	assert.Equal(user.Icon, icon)
}
