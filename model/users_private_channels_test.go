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

	upc, err := GetPrivateChannel(user1.ID, user2.ID)
	if assert.NoError(err) {
		assert.Equal(channel.ID, upc.ChannelID)
	}
}

func TestGetMember(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	user1 := mustMakeUser(t, "private-1")
	user2 := mustMakeUser(t, "private-2")
	channel := mustMakePrivateChannel(t, user1.ID, user2.ID, "privatechannel-1")

	member, err := GetPrivateChannelMembers(channel.ID)
	assert.NoError(err)
	assert.Len(member, 2)
}
