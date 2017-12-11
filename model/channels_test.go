package model

import (
	"strconv"
	"testing"
)

// 各関数のテスト>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
func TestCreate(t *testing.T) {
	beforeTest(t)
	channel := new(Channel)
	channel.CreatorID = testUserID
	channel.Name = "testChannel"
	channel.IsPublic = true

	err := channel.Create()
	if err != nil {
		t.Fatal("Failed to create channel", err)
	}

	if channel.CreatorID != testUserID {
		t.Errorf("CreatorId: want %s, acutual %s", testUserID, channel.CreatorID)
	}

	if channel.UpdaterID != testUserID {
		t.Errorf("UpdaterId: want %s, acutual %s", testUserID, channel.UpdaterID)
	}
}

func TestGetChannelList(t *testing.T) {
	beforeTest(t)
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
	beforeTest(t)
	parentChannel, err := makeChannelDetail(testUserID, "parent", "", true)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		makeChannelDetail(testUserID, "child-"+strconv.Itoa(i+1), parentChannel.ID, true)
	}

	for i := 10; i < 20; i++ {
		channel, _ := makeChannelDetail(privateUserID, "child-"+strconv.Itoa(i+1), parentChannel.ID, false)
		usersPrivateChannel := new(UsersPrivateChannel)
		usersPrivateChannel.ChannelID = channel.ID
		usersPrivateChannel.UserID = privateUserID
		usersPrivateChannel.Create()
	}

	idList, err := GetChildrenChannelIDList(testUserID, parentChannel.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(idList) != 10 {
		t.Fatalf("Children Id list length wrong: want %d, acutual %d\n", 10, len(idList))
	}

	idList, err = GetChildrenChannelIDList(privateUserID, parentChannel.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(idList) != 20 {
		t.Fatalf("Children Id list length wrong: want %d, acutual %d\n", 20, len(idList))
	}
}

func TestUpdate(t *testing.T) {
	beforeTest(t)
	channel := new(Channel)
	channel.CreatorID = testUserID
	channel.Name = "Channel"
	channel.IsPublic = true
	if err := channel.Create(); err != nil {
		t.Fatal(err)
	}

	parentChannel := new(Channel)
	parentChannel.CreatorID = testUserID
	parentChannel.Name = "Parent"
	parentChannel.IsPublic = true
	if err := parentChannel.Create(); err != nil {
		t.Fatal("Failed to create channel", err)
	}

	updaterID := CreateUUID()

	channel.UpdaterID = updaterID
	channel.Name = "Channel-updated"
	channel.ParentID = parentChannel.ID

	if err := channel.Update(); err != nil {
		t.Fatal("Failed to update ", err)
	}

	if channel.Name != "Channel-updated" {
		t.Errorf("Name: want %s, acutual %s\n", "Channel-updated", channel.Name)
	}

	if channel.CreatorID != testUserID {
		t.Errorf("CreatorId: want %s, acutual %s\n", testUserID, channel.CreatorID)
	}

	if channel.UpdaterID != updaterID {
		t.Errorf("UpdaterId: want %s, acutual %s\n", updaterID, channel.UpdaterID)
	}

	if channel.ParentID != channel.ParentID {
		t.Errorf("UpdaterId: want %s, acutual %s\n", parentChannel.ID, channel.ParentID)
	}
}

func TestExists(t *testing.T) {
	beforeTest(t)
	channel, err := makeChannelDetail(testUserID, "test", "", true)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := ExistsChannel(channel.ID)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("ok not true")
	}

	ok, err = ExistsChannel(CreateUUID())
	if err != nil {
		t.Fatal(err)
	}

	if ok {
		t.Fatal("ok not false")
	}
}

// 各関数のテスト<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

// 関数間のテスト>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
func TestCreateChildChannel(t *testing.T) {
	beforeTest(t)
	channel := new(Channel)
	channel.CreatorID = testUserID
	channel.Name = "testChannel"
	channel.IsPublic = true

	if err := channel.Create(); err != nil {
		t.Fatal("Failed to create channel", err)
	}

	childChannel := new(Channel)
	childChannel.CreatorID = testUserID
	childChannel.Name = "testChannelChild"
	childChannel.IsPublic = true
	childChannel.ParentID = channel.ID
	if err := childChannel.Create(); err != nil {
		t.Fatal("Failed to create channel", err)
	}

	if childChannel.CreatorID != testUserID {
		t.Errorf("CreatorId: want %s, acutual %s\n", testUserID, childChannel.CreatorID)
	}

	if childChannel.UpdaterID != testUserID {
		t.Errorf("UpdaterId: want %s, acutual %s\n", testUserID, childChannel.UpdaterID)
	}

	if childChannel.ParentID != channel.ID {
		t.Errorf("UpdaterId: want %s, acutual %s\n", channel.ID, childChannel.ID)
	}
}

// 関数間のテスト<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

func makeChannel(tail string) error {
	channel := new(Channel)
	channel.CreatorID = testUserID
	channel.Name = "Channel-" + tail
	channel.IsPublic = true
	return channel.Create()
}

func makeChannelDetail(creatorID, name, parentID string, isPublic bool) (*Channel, error) {
	channel := new(Channel)
	channel.CreatorID = creatorID
	channel.Name = name
	channel.ParentID = parentID
	channel.IsPublic = isPublic
	err := channel.Create()
	return channel, err
}
