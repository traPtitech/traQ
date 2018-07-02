package model

import (
	"github.com/satori/go.uuid"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsersTag_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users_tags", (&UsersTag{}).TableName())
}

func TestAddUserTag(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	tag := mustMakeTag(t, "test")
	assert.NoError(AddUserTag(user.GetUID(), tag.GetID()))
}

func TestChangeUserTagLock(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	tag := mustMakeTag(t, "test")
	require.NoError(AddUserTag(user.GetUID(), tag.GetID()))

	if assert.NoError(ChangeUserTagLock(user.GetUID(), tag.GetID(), true)) {
		tag, err := GetUserTag(user.GetUID(), tag.GetID())
		require.NoError(err)
		assert.True(tag.IsLocked)
	}

	if assert.NoError(ChangeUserTagLock(user.GetUID(), tag.GetID(), false)) {
		tag, err := GetUserTag(user.GetUID(), tag.GetID())
		require.NoError(err)
		assert.False(tag.IsLocked)
	}
}

func TestDeleteUserTag(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	tag := mustMakeTag(t, "test")
	require.NoError(AddUserTag(user.GetUID(), tag.GetID()))
	tag2 := mustMakeTag(t, "test2")
	require.NoError(AddUserTag(user.GetUID(), tag2.GetID()))

	if assert.NoError(DeleteUserTag(user.GetUID(), tag.GetID())) {
		_, err := GetUserTag(user.GetUID(), tag.GetID())
		assert.Error(err)
	}

	_, err := GetUserTag(user.GetUID(), tag2.GetID())
	assert.NoError(err)
}

func TestGetUserTagsByUserID(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	for i := 0; i < 10; i++ {
		tag := mustMakeTag(t, "test"+strconv.Itoa(i))
		require.NoError(AddUserTag(user.GetUID(), tag.GetID()))
	}

	tags, err := GetUserTagsByUserID(user.GetUID())
	if assert.NoError(err) {
		for i, v := range tags {
			if assert.NotZero(v.Tag) {
				assert.Equal("test"+strconv.Itoa(i), v.Tag.Name)
			}
		}
	}

	tags, err = GetUserTagsByUserID(uuid.Nil)
	if assert.NoError(err) {
		assert.Len(tags, 0)
	}
}

func TestGetUserTag(t *testing.T) {
	assert, require, user, _ := beforeTest(t)
	tag := mustMakeTag(t, "test")
	require.NoError(AddUserTag(user.GetUID(), tag.GetID()))

	ut, err := GetUserTag(user.GetUID(), tag.GetID())
	if assert.NoError(err) {
		assert.Equal(user.ID, ut.UserID)
		assert.Equal(tag.ID, ut.TagID)
		assert.False(ut.IsLocked)
		assert.NotZero(ut.CreatedAt)
		assert.NotZero(ut.UpdatedAt)
		if assert.NotZero(ut.Tag) {
			assert.Equal("test", ut.Tag.Name)
			assert.Equal(tag.ID, ut.Tag.ID)
			assert.False(ut.Tag.Restricted)
			assert.Empty(ut.Tag.Type)
			assert.NotZero(ut.Tag.CreatedAt)
			assert.NotZero(ut.Tag.UpdatedAt)
		}
	}

	_, err = GetUserTag(user.GetUID(), uuid.Nil)
	assert.Error(err)
}

func TestGetUserIDsByTag(t *testing.T) {
	assert, require, _, _ := beforeTest(t)

	tag := mustMakeTag(t, "test")
	for i := 0; i < 10; i++ {
		user := mustMakeUser(t, "tagTest-"+strconv.Itoa(i))
		require.NoError(AddUserTag(user.GetUID(), tag.GetID()))
	}

	ids, err := GetUserIDsByTag("test")
	if assert.NoError(err) {
		assert.Len(ids, 10)
	}

	ids, err = GetUserIDsByTag("nothing")
	if assert.NoError(err) {
		assert.Len(ids, 0)
	}
}

func TestGetUserIDsByTagID(t *testing.T) {
	assert, require, _, _ := beforeTest(t)

	tag := mustMakeTag(t, "test")
	for i := 0; i < 10; i++ {
		user := mustMakeUser(t, "tagTest-"+strconv.Itoa(i))
		require.NoError(AddUserTag(user.GetUID(), tag.GetID()))
	}

	ids, err := GetUserIDsByTagID(tag.GetID())
	if assert.NoError(err) {
		assert.Len(ids, 10)
	}

	ids, err = GetUserIDsByTagID(uuid.Nil)
	if assert.NoError(err) {
		assert.Len(ids, 0)
	}
}
