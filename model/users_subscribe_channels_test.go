package model

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUserSubscribeChannel_TableName(t *testing.T) {
	assert.Equal(t, "users_subscribe_channels", (&UserSubscribeChannel{}).TableName())
}

func TestUserSubscribeChannel_Create(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	id1 := "62e0c80d-a77a-4cee-a2c0-71eda349825b"
	id2 := "9349b372-5f73-4297-a42f-6a98d4d25454"
	channel1 := "aaefc6cc-75e5-4eee-a2f3-cae63dc3ede8"
	channel2 := "55a1f654-6fe2-4d6a-b60a-c70c8d1dedba"

	assert.NoError((&UserSubscribeChannel{UserID: id1, ChannelID: channel1}).Create())
	assert.NoError((&UserSubscribeChannel{UserID: id1, ChannelID: channel2}).Create())
	assert.NoError((&UserSubscribeChannel{UserID: id2, ChannelID: channel2}).Create())
	assert.Error((&UserSubscribeChannel{UserID: id1, ChannelID: channel2}).Create())

	l, err := db.Count(&UserSubscribeChannel{})
	require.NoError(t, err)

	assert.EqualValues(3, l)
}

func TestUserSubscribeChannel_Delete(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	id1 := "62e0c80d-a77a-4cee-a2c0-71eda349825b"
	id2 := "9349b372-5f73-4297-a42f-6a98d4d25454"
	channel1 := "aaefc6cc-75e5-4eee-a2f3-cae63dc3ede8"
	channel2 := "55a1f654-6fe2-4d6a-b60a-c70c8d1dedba"

	assert.NoError((&UserSubscribeChannel{UserID: id1, ChannelID: channel1}).Create())
	assert.NoError((&UserSubscribeChannel{UserID: id1, ChannelID: channel2}).Create())
	assert.NoError((&UserSubscribeChannel{UserID: id2, ChannelID: channel2}).Create())

	assert.NoError((&UserSubscribeChannel{UserID: id2, ChannelID: channel2}).Delete())
	l, err := db.Count(&UserSubscribeChannel{})
	require.NoError(t, err)
	assert.Equal(int64(2), l)

	assert.Error((&UserSubscribeChannel{UserID: id1}).Delete())
	assert.Error((&UserSubscribeChannel{}).Delete())
	assert.Error((&UserSubscribeChannel{ChannelID: channel1}).Delete())

	assert.NoError((&UserSubscribeChannel{UserID: id1, ChannelID: channel2}).Delete())
	l, err = db.Count(&UserSubscribeChannel{})
	require.NoError(t, err)
	assert.Equal(int64(1), l)

	assert.NoError((&UserSubscribeChannel{UserID: id1, ChannelID: channel1}).Delete())
	l, err = db.Count(&UserSubscribeChannel{})
	require.NoError(t, err)
	assert.Equal(int64(0), l)

}

func TestGetSubscribingUser(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	id1 := "62e0c80d-a77a-4cee-a2c0-71eda349825b"
	id2 := "9349b372-5f73-4297-a42f-6a98d4d25454"
	channel1 := "aaefc6cc-75e5-4eee-a2f3-cae63dc3ede8"
	channel2 := "55a1f654-6fe2-4d6a-b60a-c70c8d1dedba"

	assert.NoError((&UserSubscribeChannel{UserID: id1, ChannelID: channel1}).Create())
	assert.NoError((&UserSubscribeChannel{UserID: id1, ChannelID: channel2}).Create())
	assert.NoError((&UserSubscribeChannel{UserID: id2, ChannelID: channel2}).Create())

	arr, err := GetSubscribingUser(uuid.FromStringOrNil(channel1))
	if assert.NoError(err) {
		assert.Len(arr, 1)
	}

	arr, err = GetSubscribingUser(uuid.FromStringOrNil(channel2))
	if assert.NoError(err) {
		assert.Len(arr, 2)
	}
}
