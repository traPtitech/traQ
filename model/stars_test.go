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

	channel := mustMakeChannelDetail(t, user.GetUID(), "test", "")

	if assert.NoError(AddStar(user.GetUID(), channel.ID)) {
		count := 0
		db.Table("stars").Count(&count)
		assert.Equal(1, count)
	}
}

func TestRemoveStar(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	channel := mustMakeChannelDetail(t, user.GetUID(), "test", "")
	require.NoError(AddStar(user.GetUID(), channel.ID))
	count := 0

	if assert.NoError(RemoveStar(user.GetUID(), uuid.Nil)) {
		db.Table("stars").Count(&count)
		assert.Equal(1, count)
	}
	if assert.NoError(RemoveStar(user.GetUID(), channel.ID)) {
		db.Table("stars").Count(&count)
		require.Equal(0, count)
	}
}

func TestGetStaredChannels(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	channelCount := 5
	for i := 0; i < channelCount; i++ {
		ch := mustMakeChannelDetail(t, user.GetUID(), "test"+strconv.Itoa(i), "")
		require.NoError(AddStar(user.GetUID(), ch.ID))
	}

	ch, err := GetStaredChannels(user.GetUID())
	if assert.NoError(err) {
		assert.Len(ch, channelCount)
	}
}
