package model

import (
	"strconv"
	"testing"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "channels", (&Channel{}).TableName())
}

func TestCreatePublicChannel(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	c, err := CreatePublicChannel("", "test2", user.GetUID())
	if assert.NoError(err) {
		assert.NotEmpty(c.ID)
		assert.Equal("test2", c.Name)
		assert.Equal(user.GetUID(), c.CreatorID)
		assert.Empty(c.ParentID)
		assert.True(c.IsPublic)
		assert.True(c.IsVisible)
		assert.False(c.IsForced)
		assert.Equal(user.GetUID(), c.UpdaterID)
		assert.Empty(c.Topic)
		assert.NotZero(c.CreatedAt)
		assert.NotZero(c.UpdatedAt)
		assert.Nil(c.DeletedAt)
	}

	_, err = CreatePublicChannel("", "test2", user.GetUID())
	assert.Equal(ErrDuplicateName, err)

	_, err = CreatePublicChannel("", "ああああ", user.GetUID())
	assert.Error(err)

	c2, err := CreatePublicChannel(c.ID.String(), "Parent2", user.GetUID())
	assert.NoError(err)
	c3, err := CreatePublicChannel(c2.ID.String(), "Parent3", user.GetUID())
	assert.NoError(err)
	c4, err := CreatePublicChannel(c3.ID.String(), "Parent4", user.GetUID())
	assert.NoError(err)
	_, err = CreatePublicChannel(c3.ID.String(), "Parent4", user.GetUID())
	assert.Equal(ErrDuplicateName, err)
	c5, err := CreatePublicChannel(c4.ID.String(), "Parent5", user.GetUID())
	assert.NoError(err)
	_, err = CreatePublicChannel(c5.ID.String(), "Parent6", user.GetUID())
	assert.Equal(ErrChannelDepthLimitation, err)
}

func TestUpdateChannelTopic(t *testing.T) {
	assert, require, user, channel := beforeTest(t)

	if assert.NoError(UpdateChannelTopic(channel.ID, "test", user.GetUID())) {
		ch, err := GetChannel(channel.ID)
		require.NoError(err)
		assert.Equal("test", ch.Topic)
	}
	if assert.NoError(UpdateChannelTopic(channel.ID, "", user.GetUID())) {
		ch, err := GetChannel(channel.ID)
		require.NoError(err)
		assert.Equal("", ch.Topic)
	}
}

func TestChangeChannelName(t *testing.T) {
	assert, require, user, c1 := beforeTest(t)

	c2 := mustMakeChannelDetail(t, user.GetUID(), "test2", "")
	c3 := mustMakeChannelDetail(t, user.GetUID(), "test3", c2.ID.String())
	mustMakeChannelDetail(t, user.GetUID(), "test4", c2.ID.String())

	assert.Error(ChangeChannelName(c1.ID, "", user.GetUID()))
	assert.Error(ChangeChannelName(c1.ID, "あああ", user.GetUID()))
	assert.Error(ChangeChannelName(c1.ID, "test2", user.GetUID()))
	if assert.NoError(ChangeChannelName(c1.ID, "aiueo", user.GetUID())) {
		c, err := GetChannel(c1.ID)
		require.NoError(err)
		assert.Equal("aiueo", c.Name)
	}

	assert.Error(ChangeChannelName(c3.ID, "test4", user.GetUID()))
	if assert.NoError(ChangeChannelName(c3.ID, "test2", user.GetUID())) {
		c, err := GetChannel(c3.ID)
		require.NoError(err)
		assert.Equal("test2", c.Name)
	}
}

func TestChangeChannelParent(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	c2 := mustMakeChannelDetail(t, user.GetUID(), "test2", "")
	c3 := mustMakeChannelDetail(t, user.GetUID(), "test3", c2.ID.String())
	c4 := mustMakeChannelDetail(t, user.GetUID(), "test2", c3.ID.String())

	assert.Error(ChangeChannelParent(c4.ID, "", user.GetUID()))

	if assert.NoError(ChangeChannelParent(c3.ID, "", user.GetUID())) {
		c, err := GetChannel(c3.ID)
		require.NoError(err)
		assert.Equal("", c.ParentID)
	}
}

func TestUpdateChannelFlag(t *testing.T) {
	assert, require, user, c1 := beforeTest(t)

	flag1 := true
	flag2 := false
	if assert.NoError(UpdateChannelFlag(c1.ID, &flag2, &flag1, user.GetUID())) {
		c, err := GetChannel(c1.ID)
		require.NoError(err)
		assert.True(c.IsForced)
		assert.False(c.IsVisible)
	}
}

func TestDeleteChannel(t *testing.T) {
	assert, _, _, c1 := beforeTest(t)

	if assert.NoError(DeleteChannel(c1.ID)) {
		_, err := GetChannel(c1.ID)
		assert.Error(err)
	}
}

func TestIsChannelNamePresent(t *testing.T) {
	assert, _, user, _ := beforeTest(t)
	c2 := mustMakeChannelDetail(t, user.GetUID(), "test2", "")
	mustMakeChannelDetail(t, user.GetUID(), "test3", c2.ID.String())

	ok, err := IsChannelNamePresent("test2", "")
	if assert.NoError(err) {
		assert.True(ok)
	}

	ok, err = IsChannelNamePresent("test3", "")
	if assert.NoError(err) {
		assert.False(ok)
	}

	ok, err = IsChannelNamePresent("test3", c2.ID.String())
	if assert.NoError(err) {
		assert.True(ok)
	}

	ok, err = IsChannelNamePresent("test4", c2.ID.String())
	if assert.NoError(err) {
		assert.False(ok)
	}

}

func TestGetParentChannel(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	childChannel := mustMakeChannelDetail(t, user.GetUID(), "child", channel.ID.String())

	parent, err := GetParentChannel(childChannel.ID)
	if assert.NoError(err) {
		assert.Equal(parent.ID, channel.ID)
	}

	parent, err = GetParentChannel(channel.ID)
	if assert.NoError(err) {
		assert.Nil(parent)
	}
}

func TestChannel_Path(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	ch1 := mustMakeChannelDetail(t, user.GetUID(), "parent", "")
	ch2 := mustMakeChannelDetail(t, user.GetUID(), "child", ch1.ID.String())

	path, err := ch2.Path()
	assert.NoError(err)
	assert.Equal("#parent/child", path)

	path, err = ch1.Path()
	assert.NoError(err)
	assert.Equal("#parent", path)
}

func TestGetChannel(t *testing.T) {
	assert, _, _, channel := beforeTest(t)

	ch, err := GetChannel(channel.ID)
	if assert.NoError(err) {
		assert.Equal(channel.ID, ch.ID)
		assert.Equal(channel.Name, ch.Name)
	}

	_, err = GetChannel(uuid.Nil)
	assert.Error(err)
}

func TestGetChannelWithUserID(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	ch := mustMakeChannel(t, user.GetUID(), "getByID")

	r, err := GetChannelWithUserID(user.GetUID(), ch.ID)
	if assert.NoError(err) {
		assert.Equal(ch.Name, r.Name)
	}
	// TODO: userから見えないチャンネルの取得についてのテスト
}

func TestGetChannelByMessageID(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	message := mustMakeMessage(t, user.GetUID(), channel.ID)

	ch, err := GetChannelByMessageID(message.GetID())
	if assert.NoError(err) {
		assert.Equal(channel.ID, ch.ID)
	}

	_, err = GetChannelByMessageID(uuid.Nil)
	assert.Error(err)
}

func TestGetChannelList(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	for i := 0; i < 10; i++ {
		mustMakeChannel(t, user.GetUID(), strconv.Itoa(i))
	}

	channelList, err := GetChannelList(user.GetUID())
	if assert.NoError(err) {
		assert.Len(channelList, 10+3)
	}
}

func TestGetAllChannels(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	n := 10
	for i := 0; i < n; i++ {
		mustMakeChannel(t, user.GetUID(), strconv.Itoa(i))
	}

	chList, err := GetAllChannels()
	if assert.NoError(err) {
		assert.Equal(n+3, len(chList))
	}
}

func TestGetChannelPath(t *testing.T) {
	assert, _, _, ch := beforeTest(t)

	path, ok := GetChannelPath(ch.ID)
	assert.True(ok)
	assert.Equal("#"+ch.Name, path)
}
