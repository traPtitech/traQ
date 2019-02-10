package model

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/utils"
	"testing"
)

func TestStar_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "stars", (&Star{}).TableName())
}

// TestParallelGroup2 並列テストグループ2 競合がないようなサブテストにすること
func TestParallelGroup2(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	// AddStar
	t.Run("TestAddStar", func(t *testing.T) {
		t.Parallel()

		ch := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		user1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))

		if assert.NoError(AddStar(user1.ID, ch.ID)) {
			count := 0
			db.Model(Star{}).Where("user_id = ?", user1.ID).Count(&count)
			assert.Equal(1, count)
		}
	})

	// RemoveStar
	t.Run("TestRemoveStar", func(t *testing.T) {
		t.Parallel()

		ch := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		user1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		require.NoError(AddStar(user1.ID, ch.ID))

		count := 0
		if assert.NoError(RemoveStar(user1.ID, uuid.Nil)) {
			db.Model(Star{}).Where("user_id = ?", user1.ID).Count(&count)
			assert.Equal(1, count)
		}
		if assert.NoError(RemoveStar(user1.ID, ch.ID)) {
			db.Model(Star{}).Where("user_id = ?", user1.ID).Count(&count)
			assert.Equal(0, count)
		}
	})

	// GetStaredChannels
	t.Run("TestGetStaredChannels", func(t *testing.T) {
		t.Parallel()

		user1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		channelCount := 5
		for i := 0; i < channelCount; i++ {
			ch := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
			require.NoError(AddStar(user1.ID, ch.ID))
		}

		ch, err := GetStaredChannels(user1.ID)
		if assert.NoError(err) {
			assert.Len(ch, channelCount)
		}
	})
}
