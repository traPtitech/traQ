package model

import (
	"strconv"
	"testing"
)

func TestCreate(t *testing.T) {
	beforeTest(t)
	defer Close()

	channel := new(Channels)
	channel.CreatorId = testUserID
	channel.Name = "testChannel"
	channel.IsPublic = true

	err := channel.Create()
	if err != nil {
		t.Fatal("Failed to create channel", err)
	}

	if channel.CreatorId != testUserID {
		t.Errorf("CreatorId: want %s, acutual %s", testUserID, channel.CreatorId)
	}

	if channel.UpdaterId != testUserID {
		t.Errorf("UpdaterId: want %s, acutual %s", testUserID, channel.UpdaterId)
	}
}

func TestCreateChildChannel(t *testing.T) {
	beforeTest(t)
	defer Close()

	channel := new(Channels)
	channel.CreatorId = testUserID
	channel.Name = "testChannel"
	channel.IsPublic = true

	if err := channel.Create(); err != nil {
		t.Fatal("Failed to create channel", err)
	}

	childChannel := new(Channels)
	childChannel.CreatorId = testUserID
	childChannel.Name = "testChannelChild"
	childChannel.IsPublic = true
	childChannel.ParentId = channel.Id
	if err := childChannel.Create(); err != nil {
		t.Fatal("Failed to create channel", err)
	}

	if childChannel.CreatorId != testUserID {
		t.Errorf("CreatorId: want %s, acutual %s\n", testUserID, childChannel.CreatorId)
	}

	if childChannel.UpdaterId != testUserID {
		t.Errorf("UpdaterId: want %s, acutual %s\n", testUserID, childChannel.UpdaterId)
	}

	if childChannel.ParentId != channel.Id {
		t.Errorf("UpdaterId: want %s, acutual %s\n", channel.Id, childChannel.Id)
	}
}

func TestGetChannelList(t *testing.T) {
	beforeTest(t)
	defer Close()

	for i := 0; i < 10; i++ {
		err := makeChannel(strconv.Itoa(i))
		if err != nil {
			t.Fatal(err)
		}
	}

	channelList, err := GetChannelList(testUserID)

	if err != nil {
		t.Fatal("Failed to GetChannelList ", err)
	}

	if len(channelList) != 10 {
		t.Errorf("ChannelList length wrong: want 10, acutual %d\n", len(channelList))
	}
}

func TestUpdate(t *testing.T) {
	beforeTest(t)
	defer Close()

	channel := new(Channels)
	channel.CreatorId = testUserID
	channel.Name = "Channel"
	channel.IsPublic = true
	if err := channel.Create(); err != nil {
		t.Fatal(err)
	}

	parentChannel := new(Channels)
	parentChannel.CreatorId = testUserID
	parentChannel.Name = "Parent"
	parentChannel.IsPublic = true
	if err := parentChannel.Create(); err != nil {
		t.Fatal("Failed to create channel", err)
	}

	updaterId := CreateUUID()

	channel.UpdaterId = updaterId
	channel.Name = "Channel-updated"
	channel.ParentId = parentChannel.Id

	if err := channel.Update(); err != nil {
		t.Fatal("Failed to update ", err)
	}

	if channel.Name != "Channel-updated" {
		t.Errorf("Name: want %s, acutual %s\n", "Channel-updated", channel.Name)
	}

	if channel.CreatorId != testUserID {
		t.Errorf("CreatorId: want %s, acutual %s\n", testUserID, channel.CreatorId)
	}

	if channel.UpdaterId != updaterId {
		t.Errorf("UpdaterId: want %s, acutual %s\n", updaterId, channel.UpdaterId)
	}

	if channel.ParentId != channel.ParentId {
		t.Errorf("UpdaterId: want %s, acutual %s\n", parentChannel.Id, channel.ParentId)
	}
}

func makeChannel(tail string) error {
	channel := new(Channels)
	channel.CreatorId = testUserID
	channel.Name = "Channel-" + tail
	channel.IsPublic = true
	return channel.Create()
}
