package model

import (
	"github.com/satori/go.uuid"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsersPrivateChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users_private_channels", (&UsersPrivateChannel{}).TableName())
}

func TestAddPrivateChannelMember(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	channel := &Channel{
		ID:        CreateUUID(),
		CreatorID: user.ID,
		UpdaterID: user.ID,
		Name:      "Private-Channel",
		IsPublic:  false,
	}
	require.NoError(db.Create(channel).Error)

	po := mustMakeUser(t, "po")

	assert.NoError(AddPrivateChannelMember(channel.GetCID(), user.GetUID()))
	assert.NoError(AddPrivateChannelMember(channel.GetCID(), po.GetUID()))

	channelList, err := GetChannelList(user.GetUID())
	if assert.NoError(err) {
		assert.Len(channelList, 1+1)
	}

	channelList, err = GetChannelList(uuid.Nil)
	if assert.NoError(err) {
		assert.Len(channelList, 0+1)
	}
}

func TestGetPrivateChannel(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	user1 := mustMakeUser(t, "private-1")
	user2 := mustMakeUser(t, "private-2")
	channel := mustMakePrivateChannel(t, user1.GetUID(), user2.GetUID(), "privatechannel-1")

	upcID, err := GetPrivateChannel(user1.ID, user2.ID)
	if assert.NoError(err) {
		assert.Equal(channel.ID, upcID)
	}

	channel = mustMakePrivateChannel(t, user1.GetUID(), user1.GetUID(), "self-channel")
	upcID, err = GetPrivateChannel(user1.ID, user1.ID)
	if assert.NoError(err) {
		assert.Equal(channel.ID, upcID)
	}

	// 異常系：存在しないprivateチャンネルを取得する
	user3 := mustMakeUser(t, "private-3")
	upcID, err = GetPrivateChannel(user3.ID, user2.ID)
	if assert.Error(err) {
		assert.Equal(ErrNotFound, err)
	}

	upcID, err = GetPrivateChannel(user3.ID, user3.ID)
	if assert.Error(err) {
		assert.Equal(ErrNotFound, err)
	}
}

func TestGetPrivateChannelMembers(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	user1 := mustMakeUser(t, "private-1")
	user2 := mustMakeUser(t, "private-2")
	channel := mustMakePrivateChannel(t, user1.GetUID(), user2.GetUID(), "privatechannel-1")

	member, err := GetPrivateChannelMembers(channel.ID)
	assert.NoError(err)
	assert.Len(member, 2)
}

func TestIsUserPrivatateChannelMember(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	user1 := mustMakeUser(t, "private-1")
	user2 := mustMakeUser(t, "private-2")
	channel := mustMakePrivateChannel(t, user1.GetUID(), user2.GetUID(), "privatechannel-1")

	ok, err := IsUserPrivateChannelMember(channel.GetCID(), user1.GetUID())
	if assert.NoError(err) {
		assert.True(ok)
	}
	ok, err = IsUserPrivateChannelMember(channel.GetCID(), user2.GetUID())
	if assert.NoError(err) {
		assert.True(ok)
	}
	ok, err = IsUserPrivateChannelMember(channel.GetCID(), user.GetUID())
	if assert.NoError(err) {
		assert.False(ok)
	}

}
