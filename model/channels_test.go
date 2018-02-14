package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

// 各関数のテスト>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func TestChannel_TableName(t *testing.T) {
	assert.Equal(t, "channels", (&Channel{}).TableName())
}

func TestChannel_Create(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	assert.Error((&Channel{ID: "aaa"}).Create())
	assert.Error((&Channel{}).Create())
	assert.Error((&Channel{Name: "test"}).Create())
	assert.Error((&Channel{Name: "無効な名前"}).Create())

	c := &Channel{
		CreatorID: testUserID,
		Name:      "testChannel",
		ParentID:  "",
		IsPublic:  true,
	}
	if assert.NoError(c.Create()) {
		assert.NotEmpty(c.ID)
	}
}

func TestChannel_Exists(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	channel := mustMakeChannelDetail(t, testUserID, "test", "", true)

	checkChannel := &Channel{ID: channel.ID}
	ok, err := checkChannel.Exists(testUserID)
	if assert.NoError(err) {
		assert.True(ok)
	}

	checkChannel = &Channel{ID: CreateUUID()}
	ok, err = checkChannel.Exists(testUserID)
	if assert.NoError(err) {
		assert.False(ok)
	}
}

func TestChannel_Children(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	parentChannel := mustMakeChannelDetail(t, testUserID, "parent", "", true)

	for i := 0; i < 10; i++ {
		mustMakeChannelDetail(t, testUserID, "child-"+strconv.Itoa(i+1), parentChannel.ID, true)
	}

	for i := 10; i < 20; i++ {
		channel := mustMakeChannelDetail(t, privateUserID, "child-"+strconv.Itoa(i+1), parentChannel.ID, false)
		usersPrivateChannel := &UsersPrivateChannel{}
		usersPrivateChannel.ChannelID = channel.ID
		usersPrivateChannel.UserID = privateUserID
		usersPrivateChannel.Create()
	}

	idList, err := parentChannel.Children(testUserID)
	if assert.NoError(err) {
		assert.Len(idList, 10)
	}

	idList, err = parentChannel.Children(privateUserID)
	if assert.NoError(err) {
		assert.Len(idList, 20)
	}
}

func TestChannel_Update(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	channel := mustMakeChannelDetail(t, testUserID, "Channel", "", true)
	parentChannel := mustMakeChannelDetail(t, testUserID, "Parent", "", true)

	channel.UpdaterID = CreateUUID()
	channel.Name = "Channel-updated"
	channel.ParentID = parentChannel.ID

	assert.NoError(channel.Update())
}

func TestGetChannelList(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	for i := 0; i < 10; i++ {
		mustMakeChannel(t, strconv.Itoa(i))
	}

	channelList, err := GetChannels(testUserID)
	if assert.NoError(err) {
		assert.Len(channelList, 10)
	}
}

func TestValidateChannelName(t *testing.T) {
	assert := assert.New(t)

	okList := []string{"unko", "asjifas", "19012", "_a_", "---asjidfa---", "1-1", "jijijijijijijijijiji"}
	for _, name := range okList {
		assert.NoErrorf(validateChannelName(name), "channel name validation failed: %s", name)
	}
	ngList := []string{",.", "dajosd.dfjios", "うんこ", "てすｔ", "ｊｋ", "sadjfifjffojfosadjfisjdfosdjoifisdoifjsaoid"}
	for _, name := range ngList {
		assert.Errorf(validateChannelName(name), "channel name validation failed: %s", name)
	}
}

// 各関数のテスト<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

// 関数間のテスト>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
func TestCreateChildChannel(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	channel := &Channel{}
	channel.CreatorID = testUserID
	channel.Name = "testChannel"
	channel.IsPublic = true
	require.NoError(t, channel.Create())

	childChannel := &Channel{}
	childChannel.CreatorID = testUserID
	childChannel.Name = "testChannelChild"
	childChannel.IsPublic = true
	childChannel.ParentID = channel.ID
	assert.NoError(childChannel.Create())
}

// 関数間のテスト<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
