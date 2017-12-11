package model

import "testing"

func TestMakePrivateChannel(t *testing.T) {
	beforeTest(t)
	channel := new(Channel)
	channel.CreatorID = testUserID
	channel.Name = "Private-Channel"
	channel.IsPublic = false
	if err := channel.Create(); err != nil {
		t.Fatal(err)
	}

	po := CreateUUID()
	privilegedUser := []string{testUserID, po}

	for _, userID := range privilegedUser {
		usersPrivateChannel := new(UsersPrivateChannel)
		usersPrivateChannel.ChannelID = channel.ID
		usersPrivateChannel.UserID = userID
		usersPrivateChannel.Create()
	}

	channelList, err := GetChannelList(testUserID)

	if err != nil {
		t.Fatal("Failed to GetChannelList ", err)
	}
	if len(channelList) != 1 {
		t.Errorf("ChannelList length wrong: want 1, acutual %d\n", len(channelList))
	}

	channelList, err = GetChannelList(CreateUUID())
	if err != nil {
		t.Fatal("Failed to GetChannelList ", err)
	}
	if len(channelList) != 0 {
		t.Errorf("ChannelList length wrong: want 0, acutual %d\n", len(channelList))
	}
}
