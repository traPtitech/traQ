package model

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestUserInvisibleChananel_TabelName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users_invisible_channels", (&UserInvisibleChannel{}).TableName())
}

func TestUserInvisibleChananel_Create(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	assert.Error((&UserInvisibleChannel{}).Create())
	assert.Error((&UserInvisibleChannel{UserID: user.ID}).Create())
	assert.Error((&UserInvisibleChannel{ChannelID: channel.ID}).Create())
	assert.Error((&UserInvisibleChannel{UserID: CreateUUID()}).Create())
	assert.Error((&UserInvisibleChannel{ChannelID: CreateUUID()}).Create())

	assert.NoError((&UserInvisibleChannel{UserID: user.ID, ChannelID: channel.ID}).Create())
}

func TestUserInvisibleChananel_Exists(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	i := mustMakeInvisibleChannel(t, channel.ID, user.ID)
	ok, err := i.Exists()
	if assert.NoError(err) {
		assert.True(ok)
	}

	i = &UserInvisibleChannel{
		UserID:    CreateUUID(),
		ChannelID: CreateUUID(),
	}
	ok, err = i.Exists()
	if assert.NoError(err) {
		assert.False(ok)
	}
}

func TestUserInvisibleChananel_Delete(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	i := mustMakeInvisibleChannel(t, channel.ID, user.ID)

	ok, err := db.Get(i)
	if assert.NoError(err) {
		assert.True(ok)
	}

	assert.NoError(i.Delete())

	ok, err = db.Get(i)
	if assert.NoError(err) {
		assert.False(ok)
	}
}

func TestGetInvisibleChannelsByID(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	for i := 0; i < 5; i++ {
		c := mustMakeChannel(t, user.ID, "-"+strconv.Itoa(i))
		mustMakeInvisibleChannel(t, c.ID, user.ID)
	}

	c := mustMakeChannelDetail(t, user.ID, "visible-private", "", false)
	p := &UsersPrivateChannel{
		UserID:    user.ID,
		ChannelID: c.ID,
	}
	require.NoError(t, p.Create())

	u := mustMakeUser(t, "invisible")
	c = mustMakeChannelDetail(t, u.ID, "invisible-private", "", false)
	p = &UsersPrivateChannel{
		UserID:    u.ID,
		ChannelID: c.ID,
	}
	require.NoError(t, p.Create())

	list, err := GetInvisibleChannelsByID(user.ID)
	if assert.NoError(err) {
		assert.Equal(6, len(list))
	}
}

func TestGetVisibleChannelsByID(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	for i := 0; i < 5; i++ {
		mustMakeChannel(t, user.ID, "-"+strconv.Itoa(i))
	}

	c := mustMakeChannelDetail(t, user.ID, "visible-private", "", false)
	p := &UsersPrivateChannel{
		UserID:    user.ID,
		ChannelID: c.ID,
	}
	require.NoError(t, p.Create())

	u := mustMakeUser(t, "invisible")
	c = mustMakeChannelDetail(t, u.ID, "invisible-private", "", false)
	p = &UsersPrivateChannel{
		UserID:    u.ID,
		ChannelID: c.ID,
	}
	require.NoError(t, p.Create())

	list, err := GetVisibleChannelsByID(user.ID)
	if assert.NoError(err) {
		assert.Equal(7, len(list))
	}

}
