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

	if assert.NoError(SubscribeChannel(user.GetUID(), channel.GetCID())) {
		count := 0
		db.Model(UserSubscribeChannel{}).Count(&count)
		assert.Equal(1, count)
	}
	assert.Error(SubscribeChannel(user.GetUID(), channel.GetCID()))
}

func TestUnsubscribeChannel(t *testing.T) {
	assert, require, user1, channel1 := beforeTest(t)

	user2 := mustMakeUser(t, "user2")
	channel2 := mustMakeChannel(t, user1.ID, "test")

	require.NoError(SubscribeChannel(user1.GetUID(), channel1.GetCID()))
	require.NoError(SubscribeChannel(user1.GetUID(), channel2.GetCID()))
	require.NoError(SubscribeChannel(user2.GetUID(), channel2.GetCID()))

	if assert.NoError(UnsubscribeChannel(user2.GetUID(), channel2.GetCID())) {
		count := 0
		db.Model(UserSubscribeChannel{}).Count(&count)
		assert.Equal(2, count)
	}
	if assert.NoError(UnsubscribeChannel(user1.GetUID(), channel2.GetCID())) {
		count := 0
		db.Model(UserSubscribeChannel{}).Count(&count)
		assert.Equal(1, count)
	}
	if assert.NoError(UnsubscribeChannel(user1.GetUID(), channel1.GetCID())) {
		count := 0
		db.Model(UserSubscribeChannel{}).Count(&count)
		assert.Equal(0, count)
	}
}

func TestGetSubscribingUser(t *testing.T) {
	assert, require, user1, channel1 := beforeTest(t)

	user2 := mustMakeUser(t, "user2")
	channel2 := mustMakeChannel(t, user1.ID, "test")

	require.NoError(SubscribeChannel(user1.GetUID(), channel1.GetCID()))
	require.NoError(SubscribeChannel(user1.GetUID(), channel2.GetCID()))
	require.NoError(SubscribeChannel(user2.GetUID(), channel2.GetCID()))

	arr, err := GetSubscribingUser(uuid.FromStringOrNil(channel1.ID))
	if assert.NoError(err) {
		assert.Len(arr, 1)
	}

	arr, err = GetSubscribingUser(uuid.FromStringOrNil(channel2.ID))
	if assert.NoError(err) {
		assert.Len(arr, 2)
	}
}

func TestGetSubscribedChannels(t *testing.T) {
	assert, require, user1, channel1 := beforeTest(t)

	user2 := mustMakeUser(t, "user2")
	channel2 := mustMakeChannel(t, user1.ID, "test")

	require.NoError(SubscribeChannel(user1.GetUID(), channel1.GetCID()))
	require.NoError(SubscribeChannel(user1.GetUID(), channel2.GetCID()))
	require.NoError(SubscribeChannel(user2.GetUID(), channel2.GetCID()))

	arr, err := GetSubscribedChannels(uuid.FromStringOrNil(user1.ID))
	if assert.NoError(err) {
		assert.Len(arr, 2)
	}

	arr, err = GetSubscribedChannels(uuid.FromStringOrNil(user2.ID))
	if assert.NoError(err) {
		assert.Len(arr, 1)
	}
}
