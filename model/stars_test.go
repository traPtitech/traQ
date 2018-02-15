package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

func TestStar_TableName(t *testing.T) {
	assert.Equal(t, "stars", (&Star{}).TableName())
}

func TestStar_Create(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	channel := mustMakeChannelDetail(t, testUserID, "test", "", true)

	assert.Error((&Star{}).Create())
	assert.Error((&Star{UserID: testUserID}).Create())
	assert.Error((&Star{ChannelID: channel.ID}).Create())
	assert.NoError((&Star{UserID: testUserID, ChannelID: channel.ID}).Create())
}

func TestStar_Delete(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	channelCount := 5
	for i := 0; i < channelCount; i++ {
		ch := mustMakeChannelDetail(t, testUserID, "test"+strconv.Itoa(i), "", true)
		s := &Star{
			UserID:    testUserID,
			ChannelID: ch.ID,
		}
		require.NoError(t, s.Create())
	}

	channels, err := GetStaredChannels(testUserID)
	assert.NoError(err)

	star := &Star{
		UserID:    testUserID,
		ChannelID: channels[0].ID,
	}
	assert.NoError(star.Delete())

	channels, err = GetStaredChannels(testUserID)
	if assert.NoError(err) {
		assert.Len(channels, channelCount-1)
	}
}

func TestGetStaredChannels(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	channelCount := 5
	channel := mustMakeChannelDetail(t, testUserID, "test0", "", true)

	star := &Star{
		UserID:    testUserID,
		ChannelID: channel.ID,
	}
	require.NoError(t, star.Create())

	for i := 1; i < channelCount; i++ {
		ch := mustMakeChannelDetail(t, testUserID, "test"+strconv.Itoa(i), "", true)
		s := &Star{
			UserID:    testUserID,
			ChannelID: ch.ID,
		}
		require.NoError(t, s.Create())
	}

	_, err := GetStaredChannels("")
	assert.Error(err)

	channels, err := GetStaredChannels(testUserID)
	if assert.NoError(err) {
		assert.Len(channels, channelCount)
	}
}
