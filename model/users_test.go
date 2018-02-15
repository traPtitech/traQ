package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	password = "test"
)

func TestUser_TableName(t *testing.T) {
	assert.Equal(t, "users", (&User{}).TableName())
}

func TestUser_Create(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

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
	beforeTest(t)
	assert := assert.New(t)

	user := mustMakeUser(t, "testUser")
	assert.NoError(checkEmptyField(user))
	assert.Equal(user.Password, hashPassword(password, user.Salt))
}

func TestGetUser(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

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

func TestAuthorization(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	mustMakeUser(t, "testUser")

	checkUser := &User{
		Name: "testUser",
	}
	assert.NoError(checkUser.Authorization(password))
	assert.NoError(checkEmptyField(checkUser))
}
