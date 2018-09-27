package model

import (
	"testing"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestUserSubscribeChannel_TableName(t *testing.T) {
	assert.Equal(t, "users_subscribe_channels", (&UserSubscribeChannel{}).TableName())
}

func TestSubscribeChannel(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	if assert.NoError(SubscribeChannel(user.GetUID(), channel.ID)) {
		count := 0
		db.Model(UserSubscribeChannel{}).Count(&count)
		assert.Equal(1, count)
	}
	assert.Error(SubscribeChannel(user.GetUID(), channel.ID))
}

func TestUnsubscribeChannel(t *testing.T) {
	assert, require, user1, channel1 := beforeTest(t)

	user2 := mustMakeUser(t, "user2")
	channel2 := mustMakeChannel(t, user1.GetUID(), "test")

	require.NoError(SubscribeChannel(user1.GetUID(), channel1.ID))
	require.NoError(SubscribeChannel(user1.GetUID(), channel2.ID))
	require.NoError(SubscribeChannel(user2.GetUID(), channel2.ID))

	if assert.NoError(UnsubscribeChannel(user2.GetUID(), channel2.ID)) {
		count := 0
		db.Model(UserSubscribeChannel{}).Count(&count)
		assert.Equal(2, count)
	}
	if assert.NoError(UnsubscribeChannel(user1.GetUID(), channel2.ID)) {
		count := 0
		db.Model(UserSubscribeChannel{}).Count(&count)
		assert.Equal(1, count)
	}
	if assert.NoError(UnsubscribeChannel(user1.GetUID(), channel1.ID)) {
		count := 0
		db.Model(UserSubscribeChannel{}).Count(&count)
		assert.Equal(0, count)
	}
}

func TestGetSubscribingUser(t *testing.T) {
	assert, require, user1, channel1 := beforeTest(t)

	user2 := mustMakeUser(t, "user2")
	channel2 := mustMakeChannel(t, user1.GetUID(), "test")

	require.NoError(SubscribeChannel(user1.GetUID(), channel1.ID))
	require.NoError(SubscribeChannel(user1.GetUID(), channel2.ID))
	require.NoError(SubscribeChannel(user2.GetUID(), channel2.ID))

	arr, err := GetSubscribingUser(channel1.ID)
	if assert.NoError(err) {
		assert.Len(arr, 1)
	}

	arr, err = GetSubscribingUser(channel2.ID)
	if assert.NoError(err) {
		assert.Len(arr, 2)
	}
}

func TestGetSubscribedChannels(t *testing.T) {
	assert, require, user1, channel1 := beforeTest(t)

	user2 := mustMakeUser(t, "user2")
	channel2 := mustMakeChannel(t, user1.GetUID(), "test")

	require.NoError(SubscribeChannel(user1.GetUID(), channel1.ID))
	require.NoError(SubscribeChannel(user1.GetUID(), channel2.ID))
	require.NoError(SubscribeChannel(user2.GetUID(), channel2.ID))

	arr, err := GetSubscribedChannels(uuid.FromStringOrNil(user1.ID))
	if assert.NoError(err) {
		assert.Len(arr, 2)
	}

	arr, err = GetSubscribedChannels(uuid.FromStringOrNil(user2.ID))
	if assert.NoError(err) {
		assert.Len(arr, 1)
	}
}
