package model

import (
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
	assert.Error((&User{Name: "test", Email: "test@test.test", Password: "test"}).Create())
	assert.Error((&User{Name: "test", Email: "test@test.test", Password: "test", Salt: "test"}).Create())
	user := &User{Name: "test", Email: "test@test.test", Password: "test", Salt: "test", Icon: CreateUUID()}
	if assert.NoError(user.Create()) {
		assert.NotEmpty(user.ID)
	}
}

func TestSetPassword(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	user := mustMakeUser(t, "testSetPassword")
	assert.NoError(checkEmptyField(user))
	assert.Equal(user.Password, hashPassword(password, user.Salt))
}

func TestGetUser(t *testing.T) {
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

func TestGetUsers(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	for i := 0; i < 5; i++ {
		mustMakeUser(t, "testGetUsers-"+string(i))
	}
	users, err := GetUsers()
	assert.NoError(err)

	// traqユーザー・テストユーザーがいるので
	assert.Len(users, 5+2)
}

func TestAuthorization(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	mustMakeUser(t, "testAuthorization")

	checkUser := &User{
		Name: "testUser",
	}
	assert.NoError(checkUser.Authorization(password))
	assert.NoError(checkEmptyField(checkUser))
}
