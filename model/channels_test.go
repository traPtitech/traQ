package model

import (
	"strconv"
	"testing"
)

// 各関数のテスト>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
func TestCreate(t *testing.T) {
	beforeTest(t)
	channel, err := makeChannelDetail(testUserID, "testChannel", "", true)

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

	channelList, err := GetChannels(testUserID)

	if err != nil {
		t.Fatal("Failed to GetChannelList ", err)
	}

	if len(channelList) != 10 {
		t.Errorf("ChannelList length wrong: want 10, acutual %d\n", len(channelList))
	}
}

func TestChildren(t *testing.T) {
	beforeTest(t)
	parentChannel, err := makeChannelDetail(testUserID, "parent", "", true)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		_, err := makeChannelDetail(testUserID, "child-"+strconv.Itoa(i+1), parentChannel.ID, true)
		if err != nil {
			t.Fatal(err)
		}
	}

	for i := 10; i < 20; i++ {
		channel, _ := makeChannelDetail(privateUserID, "child-"+strconv.Itoa(i+1), parentChannel.ID, false)
		usersPrivateChannel := &UsersPrivateChannel{}
		usersPrivateChannel.ChannelID = channel.ID
		usersPrivateChannel.UserID = privateUserID
		usersPrivateChannel.Create()
	}

	idList, err := parentChannel.Children(testUserID)
	if err != nil {
		t.Fatal(err)
	}

	if len(idList) != 10 {
		t.Fatalf("Children Id list length wrong: want %d, acutual %d\n", 10, len(idList))
	}

	idList, err = parentChannel.Children(privateUserID)
	if err != nil {
		t.Fatal(err)
	}

	if len(idList) != 20 {
		t.Fatalf("Children Id list length wrong: want %d, acutual %d\n", 20, len(idList))
	}
}

func TestUpdate(t *testing.T) {
	beforeTest(t)
	channel, err := makeChannelDetail(testUserID, "Channel", "", true)
	if err != nil {
		t.Fatal(err)
	}

	parentChannel, err := makeChannelDetail(testUserID, "Parent", "", true)
	if err != nil {
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

	checkChannel := &Channel{ID: channel.ID}
	ok, err := checkChannel.Exists(testUserID)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("ok not true")
	}

	checkChannel = &Channel{ID: CreateUUID()}
	ok, err = checkChannel.Exists(testUserID)
	if err != nil {
		t.Fatal(err)
	}

	if ok {
		t.Fatal("ok not false")
	}
}

func TestValidateChannelName(t *testing.T) {
	okList := []string{"unko", "asjifas", "19012", "_a_", "---asjidfa---", "1-1", "jijijijijijijijijiji"}
	for _, name := range okList {
		if err := validateChannelName(name); err != nil {
			t.Fatalf("Validate channel name %s wrong: want true, actual false \n%s", name, err.Error())
		}
	}
	ngList := []string{",.", "dajosd.dfjios", "うんこ", "てすｔ", "ｊｋ", "sadjfifjffojfosadjfisjdfosdjoifisdoifjsaoid"}
	for _, name := range ngList {
		if err := validateChannelName(name); err == nil {
			t.Fatalf("Validate channel name %s wrong: want false, actual true", name)
		}
	}
}

// 各関数のテスト<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

// 関数間のテスト>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
func TestCreateChildChannel(t *testing.T) {
	beforeTest(t)
	channel := &Channel{}
	channel.CreatorID = testUserID
	channel.Name = "testChannel"
	channel.IsPublic = true

	if err := channel.Create(); err != nil {
		t.Fatal("Failed to create channel", err)
	}

	childChannel := &Channel{}
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
