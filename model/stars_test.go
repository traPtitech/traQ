package model

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestStar_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "stars", (&Star{}).TableName())
}

func TestAddStar(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	channel := mustMakeChannelDetail(t, user.GetUID(), "test", "", true)

	star, err := AddStar(user.GetUID(), channel.GetCID())
	if assert.NoError(err) {
		assert.Equal(channel.ID, star.ChannelID)
		assert.Equal(user.ID, star.UserID)
		assert.NotZero(star.CreatedAt)
		count := 0
		db.Table("stars").Count(&count)
		assert.Equal(1, count)
	}
}

func TestRemoveStar(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	channel := mustMakeChannelDetail(t, user.GetUID(), "test", "", true)
	_, err := AddStar(user.GetUID(), channel.GetCID())
	require.NoError(err)
	count := 0

	if assert.NoError(RemoveStar(user.GetUID(), uuid.Nil)) {
		db.Table("stars").Count(&count)
		assert.Equal(1, count)
	}
	if assert.NoError(RemoveStar(user.GetUID(), channel.GetCID())) {
		db.Table("stars").Count(&count)
		require.Equal(0, count)
	}
}

func TestGetStaredChannels(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	channelCount := 5
	for i := 0; i < channelCount; i++ {
		ch := mustMakeChannelDetail(t, user.GetUID(), "test"+strconv.Itoa(i), "", true)
		_, err := AddStar(user.GetUID(), ch.GetCID())
		require.NoError(err)
	}

	ch, err := GetStaredChannels(user.GetUID())
	if assert.Error(err) {
		assert.Len(ch, channelCount)
	}
}
