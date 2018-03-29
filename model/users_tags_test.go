package model

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsersTag_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users_tags", (&UsersTag{}).TableName())
}

func TestUsersTag_Create(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	ut := &UsersTag{UserID: user.ID}
	assert.NoError(ut.Create("全強"))
	assert.Error((&UsersTag{}).Create(""))
	assert.Error((&UsersTag{}).Create("aaa"))
}

func TestUsersTag_Update(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	tag := &UsersTag{
		UserID: user.ID,
	}
	require.NoError(t, tag.Create("pro"))

	tag.IsLocked = true
	assert.NoError(tag.Update())
}

func TestUsersTag_Delete(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	tag := &UsersTag{UserID: user.ID}
	require.NoError(t, tag.Create("全強"))
	assert.NoError(tag.Delete())
}

func TestGetUserTagsByUserID(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	// 正常系
	var tags [10]*UsersTag
	for i := 0; i < len(tags); i++ {
		tags[i] = &UsersTag{
			UserID: user.ID,
		}
		time.Sleep(1500 * time.Millisecond)
		require.NoError(t, tags[i].Create(strconv.Itoa(i)))
	}

	r, err := GetUserTagsByUserID(user.ID)
	if assert.NoError(err) {
		for i, v := range r {
			assert.Equal(tags[i].TagID, v.TagID)
		}
	}

	// 異常系
	notExistID := CreateUUID()
	empty, err := GetUserTagsByUserID(notExistID)
	if assert.NoError(err) {
		assert.Nil(empty)
	}
}

func TestGetTag(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	text := "test"
	// 正常系
	tag := &UsersTag{
		UserID: user.ID,
	}
	require.NoError(t, tag.Create(text))

	r, err := GetTag(tag.UserID, tag.TagID)
	if assert.NoError(err) {
		assert.Equal(tag.UserID, r.UserID)
		assert.Equal(tag.TagID, r.TagID)
	}

	// 異常系
	_, err = GetTag(user.ID, CreateUUID())
	assert.Error(err)
}

func TestGetUserIDsByTags(t *testing.T) {
	assert, _, _, _ := beforeTest(t)
	text := "tagTest"

	for i := 0; i < 5; i++ {
		u := mustMakeUser(t, "tagTest-"+strconv.Itoa(i))
		tag := &UsersTag{UserID: u.ID}
		require.NoError(t, tag.Create(text))
	}

	IDs, err := GetUserIDsByTags([]string{text})
	assert.NoError(err)
	assert.Len(IDs, 5)
}
