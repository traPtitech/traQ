package model

import (
	"fmt"
	"github.com/traPtitech/traQ/utils"
	"strconv"
	"testing"

	"github.com/satori/go.uuid"
)

// TestParallelGroup1 並列テストグループ1 競合がないようなサブテストにすること
func TestParallelGroup1(t *testing.T) {
	assert, require, user, channel := beforeTest(t)

	// Channel.TableName
	t.Run("TestChannel_TableName", func(t *testing.T) {
		t.Parallel()
		assert.Equal("channels", (&Channel{}).TableName())
	})

	// UpdateChannelTopic
	t.Run("TestUpdateChannelTopic", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			topic string
		}{
			{"test"},
			{""},
		}

		for i, v := range cases {
			v := v
			i := i
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				t.Parallel()
				ch := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
				if assert.NoError(UpdateChannelTopic(ch.ID, v.topic, user.GetUID())) {
					ch, err := GetChannel(ch.ID)
					require.NoError(err)
					assert.Equal(v.topic, ch.Topic)
				}
			})
		}
	})

	// UpdateChannelFlag
	t.Run("TestUpdateChannelFlag", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			flag1 bool
			flag2 bool
		}{
			{true, true},
			{true, false},
			{false, true},
			{false, false},
		}

		for i, v := range cases {
			v := v
			i := i
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				t.Parallel()
				ch := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
				if assert.NoError(UpdateChannelFlag(ch.ID, &v.flag1, &v.flag2, user.GetUID())) {
					c, err := GetChannel(ch.ID)
					require.NoError(err)
					assert.Equal(v.flag1, c.IsVisible)
					assert.Equal(v.flag2, c.IsForced)
				}
			})
		}
	})

	// GetChannelByMessageID
	t.Run("TestGetChannelByMessageID", func(t *testing.T) {
		t.Parallel()

		t.Run("Exists", func(t *testing.T) {
			t.Parallel()

			ch := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
			message := mustMakeMessage(t, user.GetUID(), ch.ID)
			ch, err := GetChannelByMessageID(message.GetID())
			if assert.NoError(err) {
				assert.Equal(ch.ID, ch.ID)
			}
		})

		t.Run("NotExists", func(t *testing.T) {
			t.Parallel()

			_, err := GetChannelByMessageID(uuid.Nil)
			assert.Error(err)
		})
	})

	// GetChannel
	t.Run("TestGetChannel", func(t *testing.T) {
		t.Parallel()

		t.Run("Exists", func(t *testing.T) {
			t.Parallel()
			ch, err := GetChannel(channel.ID)
			if assert.NoError(err) {
				assert.Equal(channel.ID, ch.ID)
				assert.Equal(channel.Name, ch.Name)
			}
		})

		t.Run("NotExists", func(t *testing.T) {
			_, err := GetChannel(uuid.Nil)
			assert.Error(err)
		})
	})

	// Channel.Path
	t.Run("TestChannel_Path", func(t *testing.T) {
		t.Parallel()

		ch1 := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		ch2 := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), ch1.ID.String())
		ch3 := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), ch2.ID.String())

		t.Run("ch1", func(t *testing.T) {
			t.Parallel()

			path, err := ch1.Path()
			assert.NoError(err)
			assert.Equal(fmt.Sprintf("#%s", ch1.Name), path)
		})

		t.Run("ch2", func(t *testing.T) {
			t.Parallel()

			path, err := ch2.Path()
			assert.NoError(err)
			assert.Equal(fmt.Sprintf("#%s/%s", ch1.Name, ch2.Name), path)
		})

		t.Run("ch3", func(t *testing.T) {
			t.Parallel()

			path, err := ch3.Path()
			assert.NoError(err)
			assert.Equal(fmt.Sprintf("#%s/%s/%s", ch1.Name, ch2.Name, ch3.Name), path)
		})
	})

	// GetChannelPath
	t.Run("TestGetChannelPath", func(t *testing.T) {
		t.Parallel()

		ch1 := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		ch2 := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), ch1.ID.String())
		ch3 := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), ch2.ID.String())

		t.Run("ch1", func(t *testing.T) {
			t.Parallel()

			path, ok := GetChannelPath(ch1.ID)
			assert.True(ok)
			assert.Equal(fmt.Sprintf("#%s", ch1.Name), path)
		})

		t.Run("ch2", func(t *testing.T) {
			t.Parallel()

			path, ok := GetChannelPath(ch2.ID)
			assert.True(ok)
			assert.Equal(fmt.Sprintf("#%s/%s", ch1.Name, ch2.Name), path)
		})

		t.Run("ch3", func(t *testing.T) {
			t.Parallel()

			path, ok := GetChannelPath(ch3.ID)
			assert.True(ok)
			assert.Equal(fmt.Sprintf("#%s/%s/%s", ch1.Name, ch2.Name, ch3.Name), path)
		})

		t.Run("NotExists", func(t *testing.T) {
			t.Parallel()

			_, ok := GetChannelPath(uuid.Nil)
			assert.False(ok)
		})
	})

	// GetParentChannel
	t.Run("TestGetParentChannel", func(t *testing.T) {
		t.Parallel()
		parentChannel := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		childChannel := mustMakeChannelDetail(t, user.GetUID(), "child", parentChannel.ID.String())

		t.Run("child", func(t *testing.T) {
			t.Parallel()

			parent, err := GetParentChannel(childChannel.ID)
			if assert.NoError(err) {
				assert.Equal(parent.ID, parentChannel.ID)
			}
		})

		t.Run("parent", func(t *testing.T) {
			t.Parallel()

			parent, err := GetParentChannel(parentChannel.ID)
			if assert.NoError(err) {
				assert.Nil(parent)
			}
		})

		t.Run("NotExists", func(t *testing.T) {
			t.Parallel()

			_, err := GetParentChannel(uuid.Nil)
			assert.Error(err)
		})
	})

	// IsChannelNamePresent
	t.Run("TestIsChannelNamePresent", func(t *testing.T) {
		t.Parallel()

		parent := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		c2 := mustMakeChannelDetail(t, user.GetUID(), "test2", parent.ID.String())
		mustMakeChannelDetail(t, user.GetUID(), "test3", c2.ID.String())

		cases := []struct {
			parentID uuid.UUID
			name     string
			expect   bool
		}{
			{parent.ID, "test2", true},
			{parent.ID, "test3", false},
			{c2.ID, "test3", true},
			{c2.ID, "test4", false},
		}

		for i, v := range cases {
			v := v
			i := i
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				t.Parallel()

				ok, err := IsChannelNamePresent(v.name, v.parentID.String())
				if assert.NoError(err) {
					assert.Equal(v.expect, ok)
				}
			})
		}
	})

	// ChangeChannelName
	t.Run("TestChangeChannelName", func(t *testing.T) {
		t.Parallel()

		parent := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		c2 := mustMakeChannelDetail(t, user.GetUID(), "test2", parent.ID.String())
		c3 := mustMakeChannelDetail(t, user.GetUID(), "test3", c2.ID.String())
		mustMakeChannelDetail(t, user.GetUID(), "test4", c2.ID.String())

		t.Run("fail", func(t *testing.T) {
			t.Parallel()

			assert.Error(ChangeChannelName(channel.ID, "", user.GetUID()))
			assert.Error(ChangeChannelName(channel.ID, "あああ", user.GetUID()))
			assert.Error(ChangeChannelName(channel.ID, "test2???", user.GetUID()))
		})

		t.Run("c2", func(t *testing.T) {
			t.Parallel()

			if assert.NoError(ChangeChannelName(c2.ID, "aiueo", user.GetUID())) {
				c, err := GetChannel(c2.ID)
				require.NoError(err)
				assert.Equal("aiueo", c.Name)
			}
		})

		t.Run("c3", func(t *testing.T) {
			t.Parallel()

			assert.Error(ChangeChannelName(c3.ID, "test4", user.GetUID()))
			if assert.NoError(ChangeChannelName(c3.ID, "test2", user.GetUID())) {
				c, err := GetChannel(c3.ID)
				require.NoError(err)
				assert.Equal("test2", c.Name)
			}
		})
	})

	// ChangeChannelParent
	t.Run("TestChangeChannelParent", func(t *testing.T) {
		t.Parallel()

		chName := utils.RandAlphabetAndNumberString(20)
		c2 := mustMakeChannelDetail(t, user.GetUID(), chName, "")
		c3 := mustMakeChannelDetail(t, user.GetUID(), "test3", c2.ID.String())
		c4 := mustMakeChannelDetail(t, user.GetUID(), chName, c3.ID.String())

		t.Run("fail", func(t *testing.T) {
			t.Parallel()

			assert.Error(ChangeChannelParent(c4.ID, "", user.GetUID()))
		})

		t.Run("success", func(t *testing.T) {
			t.Parallel()

			if assert.NoError(ChangeChannelParent(c3.ID, "", user.GetUID())) {
				c, err := GetChannel(c3.ID)
				require.NoError(err)
				assert.Equal("", c.ParentID)
			}
		})
	})

	// DeleteChannel
	t.Run("TestDeleteChannel", func(t *testing.T) {
		t.Parallel()

		c1 := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		c2 := mustMakeChannelDetail(t, user.GetUID(), "test2", c1.ID.String())
		c3 := mustMakeChannelDetail(t, user.GetUID(), "test3", c2.ID.String())
		c4 := mustMakeChannelDetail(t, user.GetUID(), "test4", c3.ID.String())

		if assert.NoError(DeleteChannel(c1.ID)) {
			t.Run("c1", func(t *testing.T) {
				t.Parallel()
				_, err := GetChannel(c1.ID)
				assert.Error(err)
			})
			t.Run("c2", func(t *testing.T) {
				t.Parallel()
				_, err := GetChannel(c2.ID)
				assert.Error(err)
			})
			t.Run("c3", func(t *testing.T) {
				t.Parallel()
				_, err := GetChannel(c3.ID)
				assert.Error(err)
			})
			t.Run("c4", func(t *testing.T) {
				t.Parallel()
				_, err := GetChannel(c4.ID)
				assert.Error(err)
			})
		}
	})

	// GetChannelWithUserID
	t.Run("TestGetChannelWithUserID", func(t *testing.T) {
		t.Parallel()

		ch := mustMakeChannelDetail(t, user.GetUID(), utils.RandAlphabetAndNumberString(20), "")
		r, err := GetChannelWithUserID(user.GetUID(), ch.ID)
		if assert.NoError(err) {
			assert.Equal(ch.Name, r.Name)
		}
		// TODO: userから見えないチャンネルの取得についてのテスト
	})
}

// TestSeriesGroup 直列テストグループ1
func TestSeriesGroup1(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	// CreatePublicChannel
	t.Run("TestCreatePublicChannel", func(t *testing.T) {
		c, err := CreatePublicChannel("", "test2", user.GetUID())
		if assert.NoError(err) {
			assert.NotEmpty(c.ID)
			assert.Equal("test2", c.Name)
			assert.Equal(user.GetUID(), c.CreatorID)
			assert.Empty(c.ParentID)
			assert.True(c.IsPublic)
			assert.True(c.IsVisible)
			assert.False(c.IsForced)
			assert.Equal(user.GetUID(), c.UpdaterID)
			assert.Empty(c.Topic)
			assert.NotZero(c.CreatedAt)
			assert.NotZero(c.UpdatedAt)
			assert.Nil(c.DeletedAt)
		}

		_, err = CreatePublicChannel("", "test2", user.GetUID())
		assert.Equal(ErrDuplicateName, err)

		_, err = CreatePublicChannel("", "ああああ", user.GetUID())
		assert.Error(err)

		c2, err := CreatePublicChannel(c.ID.String(), "Parent2", user.GetUID())
		assert.NoError(err)
		c3, err := CreatePublicChannel(c2.ID.String(), "Parent3", user.GetUID())
		assert.NoError(err)
		c4, err := CreatePublicChannel(c3.ID.String(), "Parent4", user.GetUID())
		assert.NoError(err)
		_, err = CreatePublicChannel(c3.ID.String(), "Parent4", user.GetUID())
		assert.Equal(ErrDuplicateName, err)
		c5, err := CreatePublicChannel(c4.ID.String(), "Parent5", user.GetUID())
		assert.NoError(err)
		_, err = CreatePublicChannel(c5.ID.String(), "Parent6", user.GetUID())
		assert.Equal(ErrChannelDepthLimitation, err)
	})

	// チャンネル数ここまでpublic:7+2, private:0

	// GetAllChannels
	t.Run("TestGetAllChannels", func(t *testing.T) {
		chList, err := GetAllChannels()
		if assert.NoError(err) {
			assert.Equal(7+2, len(chList))
		}
	})

	// GetChannelList
	t.Run("TestGetChannelList", func(t *testing.T) {
		channelList, err := GetChannelList(user.GetUID())
		if assert.NoError(err) {
			assert.Len(channelList, 7+2)
		}

		// TODO プライベートチャンネル
	})
}
