package model

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestStar_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "stars", (&Star{}).TableName())
}

func TestStar_Create(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	channel := mustMakeChannelDetail(t, user.ID, "test", "", true)

	assert.Error((&Star{}).Create())
	assert.Error((&Star{UserID: user.ID}).Create())
	assert.Error((&Star{ChannelID: channel.ID}).Create())
	assert.NoError((&Star{UserID: user.ID, ChannelID: channel.ID}).Create())
}

func TestStar_Delete(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	channelCount := 5
	for i := 0; i < channelCount; i++ {
		ch := mustMakeChannelDetail(t, user.ID, "test"+strconv.Itoa(i), "", true)
		s := &Star{
			UserID:    user.ID,
			ChannelID: ch.ID,
		}
		require.NoError(s.Create())
	}

	channels, err := GetStaredChannels(user.ID)
	assert.NoError(err)

	star := &Star{
		UserID:    user.ID,
		ChannelID: channels[0].ID,
	}
	assert.NoError(star.Delete())

	channels, err = GetStaredChannels(user.ID)
	if assert.NoError(err) {
		assert.Len(channels, channelCount-1)
	}
}

func TestGetStaredChannels(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	channelCount := 5
	channel := mustMakeChannelDetail(t, user.ID, "test0", "", true)

	star := &Star{
		UserID:    user.ID,
		ChannelID: channel.ID,
	}
	require.NoError(star.Create())

	for i := 1; i < channelCount; i++ {
		ch := mustMakeChannelDetail(t, user.ID, "test"+strconv.Itoa(i), "", true)
		s := &Star{
			UserID:    user.ID,
			ChannelID: ch.ID,
		}
		require.NoError(s.Create())
	}

	_, err := GetStaredChannels("")
	assert.Error(err)

	channels, err := GetStaredChannels(user.ID)
	if assert.NoError(err) {
		assert.Len(channels, channelCount)
	}
}
