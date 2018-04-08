package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsersPrivateChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users_private_channels", (&UsersPrivateChannel{}).TableName())
}

func TestMakePrivateChannel(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	channel := &Channel{}
	channel.CreatorID = user.ID
	channel.Name = "Private-Channel"
	channel.IsPublic = false
	require.NoError(channel.Create())

	po := mustMakeUser(t, "po")
	privilegedUser := []string{user.ID, po.ID}

	for _, userID := range privilegedUser {
		usersPrivateChannel := &UsersPrivateChannel{}
		usersPrivateChannel.ChannelID = channel.ID
		usersPrivateChannel.UserID = userID
		require.NoError(usersPrivateChannel.Create())
	}

	channelList, err := GetChannelList(user.ID)
	if assert.NoError(err) {
		assert.Len(channelList, 1+1)
	}

	channelList, err = GetChannelList(CreateUUID())
	if assert.NoError(err) {
		assert.Len(channelList, 0+1)
	}
}

func TestGetPrivateChannel(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	user1 := mustMakeUser(t, "private-1")
	user2 := mustMakeUser(t, "private-2")
	channel := mustMakePrivateChannel(t, user1.ID, user2.ID, "privatechannel-1")

	upcID, err := GetPrivateChannel(user1.ID, user2.ID)
	if assert.NoError(err) {
		assert.Equal(channel.ID, upcID)
	}

	channel = mustMakePrivateChannel(t, user1.ID, user1.ID, "self-channel")
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

func TestGetPrivateMember(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	user1 := mustMakeUser(t, "private-1")
	user2 := mustMakeUser(t, "private-2")
	channel := mustMakePrivateChannel(t, user1.ID, user2.ID, "privatechannel-1")

	member, err := GetPrivateChannelMembers(channel.ID)
	assert.NoError(err)
	assert.Len(member, 2)
}
