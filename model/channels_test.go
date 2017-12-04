package model

import (
	"strconv"
	"testing"
)

// 各関数のテスト>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
func TestCreate(t *testing.T) {
	BeforeTest(t)
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

func TestGetChannelList(t *testing.T) {
	BeforeTest(t)
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

func TestGetChildrenChannelIdList(t *testing.T) {
	BeforeTest(t)
	defer Close()

	parentChannel, err := makeChannelDetail(testUserID, "parent", "", true)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		makeChannelDetail(testUserID, "child-"+strconv.Itoa(i+1), parentChannel.Id, true)
	}

	for i := 10; i < 20; i++ {
		channel, _ := makeChannelDetail(privateUserID, "child-"+strconv.Itoa(i+1), parentChannel.Id, false)
		usersPrivateChannel := new(UsersPrivateChannels)
		usersPrivateChannel.ChannelId = channel.Id
		usersPrivateChannel.UserId = privateUserID
		usersPrivateChannel.Create()
	}

	idList, err := GetChildrenChannelIdList(testUserID, parentChannel.Id)
	if err != nil {
		t.Fatal(err)
	}

	if len(idList) != 10 {
		t.Fatalf("Children Id list length wrong: want %d, acutual %d\n", 10, len(idList))
	}

	idList, err = GetChildrenChannelIdList(privateUserID, parentChannel.Id)
	if err != nil {
		t.Fatal(err)
	}

	if len(idList) != 20 {
		t.Fatalf("Children Id list length wrong: want %d, acutual %d\n", 20, len(idList))
	}
}

func TestUpdate(t *testing.T) {
	BeforeTest(t)
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

// 各関数のテスト<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

// 関数間のテスト>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
func TestCreateChildChannel(t *testing.T) {
	BeforeTest(t)
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

// 関数間のテスト<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

func makeChannel(tail string) error {
	channel := new(Channels)
	channel.CreatorId = testUserID
	channel.Name = "Channel-" + tail
	channel.IsPublic = true
	return channel.Create()
}

func makeChannelDetail(creatorId, name, parentId string, isPublic bool) (*Channels, error) {
	channel := new(Channels)
	channel.CreatorId = creatorId
	channel.Name = name
	channel.ParentId = parentId
	channel.IsPublic = isPublic
	err := channel.Create()
	return channel, err
}
