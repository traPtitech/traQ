package model

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUserSubscribeChannel_TableName(t *testing.T) {
	assert.Equal(t, "users_subscribe_channels", (&UserSubscribeChannel{}).TableName())
}

func TestUserSubscribeChannel_Create(t *testing.T) {
	assert, require, user1, channel1 := beforeTest(t)

	user2 := mustMakeUser(t, "user2")
	channel2 := mustMakeChannel(t, user1.ID, "test")

	assert.NoError((&UserSubscribeChannel{UserID: user1.ID, ChannelID: channel1.ID}).Create())
	assert.NoError((&UserSubscribeChannel{UserID: user1.ID, ChannelID: channel2.ID}).Create())
	assert.NoError((&UserSubscribeChannel{UserID: user2.ID, ChannelID: channel2.ID}).Create())
	assert.Error((&UserSubscribeChannel{UserID: user1.ID, ChannelID: channel2.ID}).Create())

	l, err := db.Count(&UserSubscribeChannel{})
	require.NoError(err)

	assert.EqualValues(3, l)
}

func TestUserSubscribeChannel_Delete(t *testing.T) {
	assert, require, user1, channel1 := beforeTest(t)

	user2 := mustMakeUser(t, "user2")
	channel2 := mustMakeChannel(t, user1.ID, "test")

	assert.NoError((&UserSubscribeChannel{UserID: user1.ID, ChannelID: channel1.ID}).Create())
	assert.NoError((&UserSubscribeChannel{UserID: user1.ID, ChannelID: channel2.ID}).Create())
	assert.NoError((&UserSubscribeChannel{UserID: user2.ID, ChannelID: channel2.ID}).Create())

	assert.NoError((&UserSubscribeChannel{UserID: user2.ID, ChannelID: channel2.ID}).Delete())
	l, err := db.Count(&UserSubscribeChannel{})
	require.NoError(err)
	assert.EqualValues(2, l)

	assert.Error((&UserSubscribeChannel{UserID: user1.ID}).Delete())
	assert.Error((&UserSubscribeChannel{}).Delete())
	assert.Error((&UserSubscribeChannel{ChannelID: channel1.ID}).Delete())

	assert.NoError((&UserSubscribeChannel{UserID: user1.ID, ChannelID: channel2.ID}).Delete())
	l, err = db.Count(&UserSubscribeChannel{})
	require.NoError(err)
	assert.EqualValues(1, l)

	assert.NoError((&UserSubscribeChannel{UserID: user1.ID, ChannelID: channel1.ID}).Delete())
	l, err = db.Count(&UserSubscribeChannel{})
	require.NoError(err)
	assert.EqualValues(0, l)

}

func TestGetSubscribingUser(t *testing.T) {
	assert, _, user1, channel1 := beforeTest(t)

	user2 := mustMakeUser(t, "user2")
	channel2 := mustMakeChannel(t, user1.ID, "test")

	assert.NoError((&UserSubscribeChannel{UserID: user1.ID, ChannelID: channel1.ID}).Create())
	assert.NoError((&UserSubscribeChannel{UserID: user1.ID, ChannelID: channel2.ID}).Create())
	assert.NoError((&UserSubscribeChannel{UserID: user2.ID, ChannelID: channel2.ID}).Create())

	arr, err := GetSubscribingUser(uuid.FromStringOrNil(channel1.ID))
	if assert.NoError(err) {
		assert.Len(arr, 1)
	}

	arr, err = GetSubscribingUser(uuid.FromStringOrNil(channel2.ID))
	if assert.NoError(err) {
		assert.Len(arr, 2)
	}
}
