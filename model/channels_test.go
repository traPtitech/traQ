package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

// 各関数のテスト>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>

func TestChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "channels", (&Channel{}).TableName())
}

func TestChannel_Create(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	assert.Error((&Channel{ID: "aaa"}).Create())
	assert.Error((&Channel{}).Create())
	assert.Error((&Channel{Name: "test"}).Create())
	assert.Error((&Channel{Name: "無効な名前"}).Create())

	c := &Channel{
		CreatorID: user.ID,
		Name:      "testChannel",
		ParentID:  "",
		IsPublic:  true,
	}
	if assert.NoError(c.Create()) {
		assert.NotEmpty(c.ID)
	}
}

func TestChannel_Exists(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	checkChannel := &Channel{ID: channel.ID}
	ok, err := checkChannel.Exists(user.ID)
	if assert.NoError(err) {
		assert.True(ok)
	}

	checkChannel = &Channel{ID: CreateUUID()}
	ok, err = checkChannel.Exists(user.ID)
	if assert.NoError(err) {
		assert.False(ok)
	}
}

func TestChannel_Children(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	privateUserID := mustMakeUser(t, "privateuser").ID

	for i := 0; i < 10; i++ {
		mustMakeChannelDetail(t, user.ID, "child-"+strconv.Itoa(i+1), channel.ID, true)
	}

	for i := 10; i < 20; i++ {
		channel := mustMakeChannelDetail(t, user.ID, "child-"+strconv.Itoa(i+1), channel.ID, false)
		usersPrivateChannel := &UsersPrivateChannel{}
		usersPrivateChannel.ChannelID = channel.ID
		usersPrivateChannel.UserID = privateUserID
		usersPrivateChannel.Create()
	}

	idList, err := channel.Children(user.ID)
	if assert.NoError(err) {
		assert.Len(idList, 10)
	}

	idList, err = channel.Children(privateUserID)
	if assert.NoError(err) {
		assert.Len(idList, 20)
	}
}

func TestChannel_Update(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	parentChannel := mustMakeChannelDetail(t, user.ID, "Parent", "", true)

	channel.UpdaterID = user.ID
	channel.Name = "Channel-updated"
	channel.ParentID = parentChannel.ID

	assert.NoError(channel.Update())
}

func TestGetChannelList(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	for i := 0; i < 10; i++ {
		mustMakeChannel(t, user.ID, strconv.Itoa(i))
	}

	channelList, err := GetChannels(user.ID)
	if assert.NoError(err) {
		assert.Len(channelList, 10+1)
	}
}

func TestValidateChannelName(t *testing.T) {
	t.Parallel()
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
	assert, _, user, _ := beforeTest(t)

	channel := &Channel{}
	channel.CreatorID = user.ID
	channel.Name = "testChannel"
	channel.IsPublic = true
	require.NoError(t, channel.Create())

	childChannel := &Channel{}
	childChannel.CreatorID = user.ID
	childChannel.Name = "testChannelChild"
	childChannel.IsPublic = true
	childChannel.ParentID = channel.ID
	assert.NoError(childChannel.Create())
}

// 関数間のテスト<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
