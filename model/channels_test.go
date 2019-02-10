package model

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/utils"
	"strconv"
	"testing"

	"github.com/satori/go.uuid"
)

func TestChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "channels", (&Channel{}).TableName())
}

func TestUsersPrivateChannel_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users_private_channels", (&UsersPrivateChannel{}).TableName())
}

func TestUserSubscribeChannel_TableName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "users_subscribe_channels", (&UserSubscribeChannel{}).TableName())
}

// TestParallelGroup1 並列テストグループ1 競合がないようなサブテストにすること
func TestParallelGroup1(t *testing.T) {
	assert, require, user, channel := beforeTest(t)

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
				ch := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
				if assert.NoError(UpdateChannelTopic(ch.ID, v.topic, user.ID)) {
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
				ch := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
				if assert.NoError(UpdateChannelFlag(ch.ID, &v.flag1, &v.flag2, user.ID)) {
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

			ch := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
			message := mustMakeMessage(t, user.ID, ch.ID)
			ch, err := GetChannelByMessageID(message.ID)
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

		ch1 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		ch2 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), ch1.ID.String())
		ch3 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), ch2.ID.String())

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

		ch1 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		ch2 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), ch1.ID.String())
		ch3 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), ch2.ID.String())

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
		parentChannel := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		childChannel := mustMakeChannelDetail(t, user.ID, "child", parentChannel.ID.String())

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

		parent := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		c2 := mustMakeChannelDetail(t, user.ID, "test2", parent.ID.String())
		mustMakeChannelDetail(t, user.ID, "test3", c2.ID.String())

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

		parent := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		c2 := mustMakeChannelDetail(t, user.ID, "test2", parent.ID.String())
		c3 := mustMakeChannelDetail(t, user.ID, "test3", c2.ID.String())
		mustMakeChannelDetail(t, user.ID, "test4", c2.ID.String())

		t.Run("fail", func(t *testing.T) {
			t.Parallel()

			assert.Error(ChangeChannelName(channel.ID, "", user.ID))
			assert.Error(ChangeChannelName(channel.ID, "あああ", user.ID))
			assert.Error(ChangeChannelName(channel.ID, "test2???", user.ID))
		})

		t.Run("c2", func(t *testing.T) {
			t.Parallel()

			if assert.NoError(ChangeChannelName(c2.ID, "aiueo", user.ID)) {
				c, err := GetChannel(c2.ID)
				require.NoError(err)
				assert.Equal("aiueo", c.Name)
			}
		})

		t.Run("c3", func(t *testing.T) {
			t.Parallel()

			assert.Error(ChangeChannelName(c3.ID, "test4", user.ID))
			if assert.NoError(ChangeChannelName(c3.ID, "test2", user.ID)) {
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
		c2 := mustMakeChannelDetail(t, user.ID, chName, "")
		c3 := mustMakeChannelDetail(t, user.ID, "test3", c2.ID.String())
		c4 := mustMakeChannelDetail(t, user.ID, chName, c3.ID.String())

		t.Run("fail", func(t *testing.T) {
			t.Parallel()

			assert.Error(ChangeChannelParent(c4.ID, "", user.ID))
		})

		t.Run("success", func(t *testing.T) {
			t.Parallel()

			if assert.NoError(ChangeChannelParent(c3.ID, "", user.ID)) {
				c, err := GetChannel(c3.ID)
				require.NoError(err)
				assert.Equal("", c.ParentID)
			}
		})
	})

	// DeleteChannel
	t.Run("TestDeleteChannel", func(t *testing.T) {
		t.Parallel()

		c1 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		c2 := mustMakeChannelDetail(t, user.ID, "test2", c1.ID.String())
		c3 := mustMakeChannelDetail(t, user.ID, "test3", c2.ID.String())
		c4 := mustMakeChannelDetail(t, user.ID, "test4", c3.ID.String())

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

		ch := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		r, err := GetChannelWithUserID(user.ID, ch.ID)
		if assert.NoError(err) {
			assert.Equal(ch.Name, r.Name)
		}
		// TODO: userから見えないチャンネルの取得についてのテスト
	})

	// GetChildrenChannelIDs
	t.Run("TestGetChildrenChannelIDs", func(t *testing.T) {
		t.Parallel()

		c1 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		c2 := mustMakeChannelDetail(t, user.ID, "child1", c1.ID.String())
		c3 := mustMakeChannelDetail(t, user.ID, "child2", c2.ID.String())
		c4 := mustMakeChannelDetail(t, user.ID, "child3", c2.ID.String())

		cases := []struct {
			name   string
			ch     uuid.UUID
			expect []uuid.UUID
		}{
			{"c1", c1.ID, []uuid.UUID{c2.ID}},
			{"c2", c2.ID, []uuid.UUID{c3.ID, c4.ID}},
			{"c3", c3.ID, []uuid.UUID{}},
			{"c4", c4.ID, []uuid.UUID{}},
		}

		for _, v := range cases {
			v := v
			t.Run(v.name, func(t *testing.T) {
				t.Parallel()

				ids, err := GetChildrenChannelIDs(v.ch)
				if assert.NoError(err) {
					assert.ElementsMatch(ids, v.expect)
				}
			})
		}
	})

	// GetDescendantChannelIDs
	t.Run("TestGetDescendantChannelIDs", func(t *testing.T) {
		t.Parallel()

		c1 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		c2 := mustMakeChannelDetail(t, user.ID, "child1", c1.ID.String())
		c3 := mustMakeChannelDetail(t, user.ID, "child2", c2.ID.String())
		c4 := mustMakeChannelDetail(t, user.ID, "child3", c2.ID.String())
		c5 := mustMakeChannelDetail(t, user.ID, "child4", c3.ID.String())

		cases := []struct {
			name   string
			ch     uuid.UUID
			expect []uuid.UUID
		}{
			{"c1", c1.ID, []uuid.UUID{c2.ID, c3.ID, c4.ID, c5.ID}},
			{"c2", c2.ID, []uuid.UUID{c3.ID, c4.ID, c5.ID}},
			{"c3", c3.ID, []uuid.UUID{c5.ID}},
			{"c4", c4.ID, []uuid.UUID{}},
			{"c5", c5.ID, []uuid.UUID{}},
		}

		for _, v := range cases {
			v := v
			t.Run(v.name, func(t *testing.T) {
				t.Parallel()

				ids, err := GetDescendantChannelIDs(v.ch)
				if assert.NoError(err) {
					assert.ElementsMatch(ids, v.expect)
				}
			})
		}
	})

	// GetAscendantChannelIDs
	t.Run("TestGetAscendantChannelIDs", func(t *testing.T) {
		t.Parallel()

		c1 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		c2 := mustMakeChannelDetail(t, user.ID, "child1", c1.ID.String())
		c3 := mustMakeChannelDetail(t, user.ID, "child2", c2.ID.String())
		c4 := mustMakeChannelDetail(t, user.ID, "child3", c2.ID.String())
		c5 := mustMakeChannelDetail(t, user.ID, "child4", c3.ID.String())

		cases := []struct {
			name   string
			ch     uuid.UUID
			expect []uuid.UUID
		}{
			{"c1", c1.ID, []uuid.UUID{}},
			{"c2", c2.ID, []uuid.UUID{c1.ID}},
			{"c3", c3.ID, []uuid.UUID{c1.ID, c2.ID}},
			{"c4", c4.ID, []uuid.UUID{c1.ID, c2.ID}},
			{"c5", c5.ID, []uuid.UUID{c1.ID, c2.ID, c3.ID}},
		}

		for _, v := range cases {
			v := v
			t.Run(v.name, func(t *testing.T) {
				t.Parallel()

				ids, err := GetAscendantChannelIDs(v.ch)
				if assert.NoError(err) {
					assert.ElementsMatch(ids, v.expect)
				}
			})
		}
	})

	// GetChannelDepth
	t.Run("TestGetChannelDepth", func(t *testing.T) {
		t.Parallel()

		c1 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		c2 := mustMakeChannelDetail(t, user.ID, "child1", c1.ID.String())
		c3 := mustMakeChannelDetail(t, user.ID, "child2", c2.ID.String())
		c4 := mustMakeChannelDetail(t, user.ID, "child3", c2.ID.String())
		c5 := mustMakeChannelDetail(t, user.ID, "child4", c3.ID.String())

		cases := []struct {
			name string
			ch   uuid.UUID
			num  int
		}{
			{"c1", c1.ID, 4},
			{"c2", c2.ID, 3},
			{"c3", c3.ID, 2},
			{"c4", c4.ID, 1},
			{"c5", c5.ID, 1},
		}

		for _, v := range cases {
			v := v
			t.Run(v.name, func(t *testing.T) {
				t.Parallel()

				d, err := GetChannelDepth(v.ch)
				if assert.NoError(err) {
					assert.Equal(v.num, d)
				}
			})
		}
	})

	// GetPrivateChannelMembers
	t.Run("TestGetPrivateChannelMembers", func(t *testing.T) {
		t.Parallel()

		user1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		user2 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		ch := mustMakePrivateChannel(t, utils.RandAlphabetAndNumberString(20), []uuid.UUID{user1.ID, user2.ID})

		member, err := GetPrivateChannelMembers(ch.ID)
		if assert.NoError(err) {
			assert.Len(member, 2)
		}
	})

	// IsUserPrivateChannelMember
	t.Run("TestIsUserPrivateChannelMember", func(t *testing.T) {
		t.Parallel()

		user1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		user2 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		ch := mustMakePrivateChannel(t, utils.RandAlphabetAndNumberString(20), []uuid.UUID{user1.ID, user2.ID})

		cases := []struct {
			name   string
			user   uuid.UUID
			expect bool
		}{
			{"user1", user1.ID, true},
			{"user2", user2.ID, true},
			{"user", user.ID, false},
		}

		for _, v := range cases {
			v := v
			t.Run(v.name, func(t *testing.T) {
				t.Parallel()

				ok, err := IsUserPrivateChannelMember(ch.ID, v.user)
				if assert.NoError(err) {
					assert.Equal(v.expect, ok)
				}
			})
		}
	})

	// SubscribeChannel
	t.Run("TestSubscribeChannel", func(t *testing.T) {
		t.Parallel()

		user1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		ch := mustMakeChannelDetail(t, user1.ID, utils.RandAlphabetAndNumberString(20), "")
		if assert.NoError(SubscribeChannel(user1.ID, ch.ID)) {
			count := 0
			db.Model(UserSubscribeChannel{}).Where(&UserSubscribeChannel{UserID: user1.ID}).Count(&count)
			assert.Equal(1, count)
		}
		assert.Error(SubscribeChannel(user1.ID, ch.ID))
	})

	// UnsubscribeChannel
	t.Run("TestUnsubscribeChannel", func(t *testing.T) {
		t.Parallel()

		user1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		user2 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		ch1 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		ch2 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		require.NoError(SubscribeChannel(user1.ID, ch1.ID))
		require.NoError(SubscribeChannel(user1.ID, ch2.ID))
		require.NoError(SubscribeChannel(user2.ID, ch2.ID))

		cases := []struct {
			name   string
			user   uuid.UUID
			ch     uuid.UUID
			expect int
		}{
			{"user2-channel2", user2.ID, ch2.ID, 2},
			{"user1-channel2", user1.ID, ch2.ID, 1},
			{"user1-channel1", user1.ID, ch1.ID, 0},
		}

		for _, v := range cases {
			v := v
			t.Run(v.name, func(t *testing.T) {
				if assert.NoError(UnsubscribeChannel(v.user, v.ch)) {
					count := 0
					db.Model(UserSubscribeChannel{}).Where("user_id IN (?, ?)", user1.ID, user2.ID).Count(&count)
					assert.Equal(v.expect, count)
				}
			})
		}
	})

	// GetSubscribingUser
	t.Run("TestGetSubscribingUser", func(t *testing.T) {
		t.Parallel()

		user1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		user2 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		ch1 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		ch2 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		require.NoError(SubscribeChannel(user1.ID, ch1.ID))
		require.NoError(SubscribeChannel(user1.ID, ch2.ID))
		require.NoError(SubscribeChannel(user2.ID, ch2.ID))

		cases := []struct {
			name   string
			ch     uuid.UUID
			expect int
		}{
			{"ch1", ch1.ID, 1},
			{"ch2", ch2.ID, 2},
		}

		for _, v := range cases {
			v := v
			t.Run(v.name, func(t *testing.T) {
				t.Parallel()

				arr, err := GetSubscribingUser(v.ch)
				if assert.NoError(err) {
					assert.Len(arr, v.expect)
				}
			})
		}
	})

	// GetSubscribedChannels
	t.Run("TestGetSubscribedChannels", func(t *testing.T) {
		t.Parallel()

		user1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		user2 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		ch1 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		ch2 := mustMakeChannelDetail(t, user.ID, utils.RandAlphabetAndNumberString(20), "")
		require.NoError(SubscribeChannel(user1.ID, ch1.ID))
		require.NoError(SubscribeChannel(user1.ID, ch2.ID))
		require.NoError(SubscribeChannel(user2.ID, ch2.ID))

		cases := []struct {
			name   string
			user   uuid.UUID
			expect int
		}{
			{"user1", user1.ID, 2},
			{"user2", user2.ID, 1},
		}

		for _, v := range cases {
			v := v
			t.Run(v.name, func(t *testing.T) {
				t.Parallel()

				arr, err := GetSubscribedChannels(v.user)
				if assert.NoError(err) {
					assert.Len(arr, v.expect)
				}
			})
		}
	})
}

// TestSeriesGroup1 直列テストグループ1
func TestSeriesGroup1(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	// CreatePublicChannel
	t.Run("TestCreatePublicChannel", func(t *testing.T) {
		c, err := CreatePublicChannel("", "test2", user.ID)
		if assert.NoError(err) {
			assert.NotEmpty(c.ID)
			assert.Equal("test2", c.Name)
			assert.Equal(user.ID, c.CreatorID)
			assert.Empty(c.ParentID)
			assert.True(c.IsPublic)
			assert.True(c.IsVisible)
			assert.False(c.IsForced)
			assert.Equal(user.ID, c.UpdaterID)
			assert.Empty(c.Topic)
			assert.NotZero(c.CreatedAt)
			assert.NotZero(c.UpdatedAt)
			assert.Nil(c.DeletedAt)
		}

		_, err = CreatePublicChannel("", "test2", user.ID)
		assert.Equal(ErrDuplicateName, err)

		_, err = CreatePublicChannel("", "ああああ", user.ID)
		assert.Error(err)

		c2, err := CreatePublicChannel(c.ID.String(), "Parent2", user.ID)
		assert.NoError(err)
		c3, err := CreatePublicChannel(c2.ID.String(), "Parent3", user.ID)
		assert.NoError(err)
		c4, err := CreatePublicChannel(c3.ID.String(), "Parent4", user.ID)
		assert.NoError(err)
		_, err = CreatePublicChannel(c3.ID.String(), "Parent4", user.ID)
		assert.Equal(ErrDuplicateName, err)
		c5, err := CreatePublicChannel(c4.ID.String(), "Parent5", user.ID)
		assert.NoError(err)
		_, err = CreatePublicChannel(c5.ID.String(), "Parent6", user.ID)
		assert.Equal(ErrChannelDepthLimitation, err)
	})

	// チャンネル数ここまでpublic:1+5, private:0

	// GetAllChannels
	t.Run("TestGetAllChannels", func(t *testing.T) {
		chList, err := GetAllChannels()
		if assert.NoError(err) {
			assert.Equal(6, len(chList))
		}
	})

	// GetChannelList
	t.Run("TestGetChannelList", func(t *testing.T) {
		channelList, err := GetChannelList(user.ID)
		if assert.NoError(err) {
			assert.Len(channelList, 6)
		}

		// TODO プライベートチャンネル
	})

	// AddPrivateChannelMember
	t.Run("TestAddPrivateChannelMember", func(t *testing.T) {
		channel := &Channel{
			CreatorID: user.ID,
			UpdaterID: user.ID,
			Name:      utils.RandAlphabetAndNumberString(20),
			IsPublic:  false,
		}
		require.NoError(db.Create(channel).Error)

		po := mustMakeUser(t, "po")

		assert.NoError(AddPrivateChannelMember(channel.ID, user.ID))
		assert.NoError(AddPrivateChannelMember(channel.ID, po.ID))

		channelList, err := GetChannelList(user.ID)
		if assert.NoError(err) {
			assert.Len(channelList, 6+1)
		}

		channelList, err = GetChannelList(uuid.Nil)
		if assert.NoError(err) {
			assert.Len(channelList, 6)
		}
	})
}
