package model

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "channels", (&Channel{}).TableName())
}

func TestChannel_IsDMChannel(t *testing.T) {
	t.Parallel()
	assert.False(t, (&Channel{ParentID: uuid.Nil}).IsDMChannel())
	assert.True(t, (&Channel{ParentID: dmChannelRootUUID}).IsDMChannel())
}

func TestUsersPrivateChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users_private_channels", (&UsersPrivateChannel{}).TableName())
}

func TestUserSubscribeChannel_TableName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "users_subscribe_channels", (&UserSubscribeChannel{}).TableName())
}

func TestDMChannelMapping_TableName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "dm_channel_mappings", (&DMChannelMapping{}).TableName())
}

func TestChannelEventType_String(t *testing.T) {
	t.Parallel()

	assert.EqualValues(t, "a", ChannelEventType("a").String())
}

func TestChannelEventDetail_Value(t *testing.T) {
	t.Parallel()

	d := ChannelEventDetail{"a": "test", "b": 123.4, "c": []interface{}{"1", "2", "4"}}

	v, err := d.Value()
	assert.NoError(t, err)
	j := ChannelEventDetail{}
	assert.NoError(t, json.Unmarshal([]byte(v.(string)), &j))
	assert.EqualValues(t, d, j)
}

func TestChannelEventDetail_Scan(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		ced := ChannelEventDetail{}
		assert.NoError(t, ced.Scan(nil))
		assert.EqualValues(t, ChannelEventDetail{}, ced)
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()

		ced := ChannelEventDetail{}
		assert.NoError(t, ced.Scan(`{"a":1.2,"b":"c","d":["e","f"]}`))
		assert.EqualValues(t,
			ChannelEventDetail{"a": 1.2, "b": "c", "d": []interface{}{"e", "f"}},
			ced,
		)
	})

	t.Run("[]byte", func(t *testing.T) {
		t.Parallel()

		ced := ChannelEventDetail{}
		assert.NoError(t, ced.Scan([]byte(`{"a":1.2,"b":"c","d":["e","f"]}`)))
		assert.EqualValues(t,
			ChannelEventDetail{"a": 1.2, "b": "c", "d": []interface{}{"e", "f"}},
			ced,
		)
	})

	t.Run("other", func(t *testing.T) {
		t.Parallel()

		ced := ChannelEventDetail{}
		assert.Error(t, ced.Scan(123))
	})
}

func TestChannelEvent_TableName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "channel_events", (&ChannelEvent{}).TableName())
}
