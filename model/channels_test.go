package model

import (
	"strconv"
	"testing"

	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
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
		Name:      "testChannel2",
		ParentID:  "",
		IsPublic:  true,
	}
	if assert.NoError(c.Create()) {
		assert.NotEmpty(c.ID)
	}

	c2 := &Channel{
		CreatorID: user.ID,
		Name:      "testChannel2",
		ParentID:  "",
		IsPublic:  true,
	}
	assert.Error(c2.Create())

	// 層の改装制限に関するテスト

	c2 = &Channel{
		CreatorID: user.ID,
		Name:      "Parent2",
		ParentID:  c.ID,
		IsPublic:  true,
	}
	assert.NoError(c2.Create())
	c3 := &Channel{
		CreatorID: user.ID,
		Name:      "Parent3",
		ParentID:  c2.ID,
		IsPublic:  true,
	}
	assert.NoError(c3.Create())
	c4 := &Channel{
		CreatorID: user.ID,
		Name:      "Parent4",
		ParentID:  c3.ID,
		IsPublic:  true,
	}
	assert.NoError(c4.Create())
	c5 := &Channel{
		CreatorID: user.ID,
		Name:      "Parent5",
		ParentID:  c4.ID,
		IsPublic:  true,
	}
	assert.NoError(c5.Create())
	c6 := &Channel{
		CreatorID: user.ID,
		Name:      "TooDeepChannel",
		ParentID:  c5.ID,
		IsPublic:  true,
	}
	if err := c6.Create(); err != nil {
		assert.Equal(ErrChannelPathDepth, err)
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

func TestChannel_Update(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	parentChannel := mustMakeChannelDetail(t, user.ID, "Parent", "", true)

	channel.UpdaterID = user.ID
	channel.Name = "Channel-updated"
	channel.Topic = "aaaa"
	channel.ParentID = parentChannel.ID
	assert.NoError(channel.Update())

	channel.Topic = ""
	assert.NoError(channel.Update())
	var topic string
	if ok, err := db.Table(channel).ID(channel.ID).Cols("topic").Get(&topic); assert.True(ok) && assert.NoError(err) {
		assert.Empty(topic)
	}
}

func TestChannel_Parent(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	childChannel := mustMakeChannelDetail(t, user.ID, "child", channel.ID, true)

	parent, err := childChannel.Parent()
	assert.NoError(err)
	assert.Equal(parent.ID, channel.ID)

	_, err = channel.Parent()
	assert.NoError(err)
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

func TestChannel_Path(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	ch1 := mustMakeChannelDetail(t, user.ID, "parent", "", true)
	ch2 := mustMakeChannelDetail(t, user.ID, "child", ch1.ID, true)

	path, err := ch2.Path()
	assert.NoError(err)
	assert.Equal("#parent/child", path)

	path, err = ch1.Path()
	assert.NoError(err)
	assert.Equal("#parent", path)
}

func TestGetChannelByID(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	ch := mustMakeChannel(t, user.ID, "getByID")

	r, err := GetChannelByID(user.ID, ch.ID)
	assert.NoError(err)
	assert.Equal(ch.Name, r.Name)
	// TODO: userから見えないチャンネルの取得についてのテスト
}

func TestGetChannelList(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	for i := 0; i < 10; i++ {
		mustMakeChannel(t, user.ID, strconv.Itoa(i))
	}

	channelList, err := GetChannelList(user.ID)
	if assert.NoError(err) {
		assert.Len(channelList, 10+1)
	}
}

func TestGetAllChannels(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	n := 10
	for i := 0; i < n; i++ {
		mustMakeChannel(t, user.ID, strconv.Itoa(i))
	}

	chList, err := GetAllChannels()
	assert.NoError(err)
	assert.Equal(n+1, len(chList)) // beforeTest(t)内で一つchannelが生成されているため+1
}

func TestGetChannelPath(t *testing.T) {
	assert, _, _, ch := beforeTest(t)

	path, ok := GetChannelPath(uuid.FromStringOrNil(ch.ID))
	assert.True(ok)
	assert.Equal("#"+ch.Name, path)
}

// 各関数のテスト<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<

// 関数間のテスト>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
func TestCreateChildChannel(t *testing.T) {
	assert, _, user, channel := beforeTest(t)

	childChannel := &Channel{}
	childChannel.CreatorID = user.ID
	childChannel.Name = "testChannelChild"
	childChannel.IsPublic = true
	childChannel.ParentID = channel.ID
	assert.NoError(childChannel.Create())
}

// 関数間のテスト<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
