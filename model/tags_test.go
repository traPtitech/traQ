package model

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTag_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "tags", (&Tag{}).TableName())
}

func TestUsersTag_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "users_tags", (&UsersTag{}).TableName())
}

// TestParallelGroup5 並列テストグループ5 競合がないようなサブテストにすること
func TestParallelGroup5(t *testing.T) {
	assert, require, _, _ := beforeTest(t)

	// AddUserTag
	t.Run("TestAddUserTag", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		tag := mustMakeTag(t, utils.RandAlphabetAndNumberString(20))
		assert.NoError(AddUserTag(user.GetUID(), tag.ID))
	})

	// ChangeUserTagLock
	t.Run("TestChangeUserTagLock", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		tag := mustMakeTag(t, utils.RandAlphabetAndNumberString(20))
		require.NoError(AddUserTag(user.GetUID(), tag.ID))

		if assert.NoError(ChangeUserTagLock(user.GetUID(), tag.ID, true)) {
			tag, err := GetUserTag(user.GetUID(), tag.ID)
			require.NoError(err)
			assert.True(tag.IsLocked)
		}

		if assert.NoError(ChangeUserTagLock(user.GetUID(), tag.ID, false)) {
			tag, err := GetUserTag(user.GetUID(), tag.ID)
			require.NoError(err)
			assert.False(tag.IsLocked)
		}
	})

	// DeleteUserTag
	t.Run("TestDeleteUserTag", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		tag := mustMakeTag(t, utils.RandAlphabetAndNumberString(20))
		require.NoError(AddUserTag(user.GetUID(), tag.ID))
		tag2 := mustMakeTag(t, utils.RandAlphabetAndNumberString(20))
		require.NoError(AddUserTag(user.GetUID(), tag2.ID))

		if assert.NoError(DeleteUserTag(user.GetUID(), tag.ID)) {
			_, err := GetUserTag(user.GetUID(), tag.ID)
			assert.Error(err)
		}

		_, err := GetUserTag(user.GetUID(), tag2.ID)
		assert.NoError(err)
	})

	// GetUserTagsByUserID
	t.Run("TestGetUserTagsByUserID", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		var createdTags []string
		for i := 0; i < 10; i++ {
			tag := mustMakeTag(t, utils.RandAlphabetAndNumberString(20))
			require.NoError(AddUserTag(user.GetUID(), tag.ID))
			createdTags = append(createdTags, tag.Name)
		}

		t.Run("has", func(t *testing.T) {
			t.Parallel()

			tags, err := GetUserTagsByUserID(user.GetUID())
			if assert.NoError(err) {
				temp := make([]string, len(tags))
				for i, v := range tags {
					temp[i] = v.Tag.Name
				}
				assert.ElementsMatch(temp, createdTags)
			}
		})

		t.Run("hasno", func(t *testing.T) {
			t.Parallel()

			tags, err := GetUserTagsByUserID(uuid.Nil)
			if assert.NoError(err) {
				assert.Empty(tags)
			}
		})
	})

	// GetUserTag
	t.Run("TestGetUserTag", func(t *testing.T) {
		t.Parallel()

		user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
		tag := mustMakeTag(t, utils.RandAlphabetAndNumberString(20))
		require.NoError(AddUserTag(user.GetUID(), tag.ID))

		t.Run("found", func(t *testing.T) {
			t.Parallel()

			ut, err := GetUserTag(user.GetUID(), tag.ID)
			if assert.NoError(err) {
				assert.Equal(user.ID, ut.UserID.String())
				assert.Equal(tag.ID, ut.TagID)
				assert.False(ut.IsLocked)
				assert.NotZero(ut.CreatedAt)
				assert.NotZero(ut.UpdatedAt)
				if assert.NotZero(ut.Tag) {
					assert.Equal(tag.Name, ut.Tag.Name)
					assert.Equal(tag.ID, ut.Tag.ID)
					assert.False(ut.Tag.Restricted)
					assert.Empty(ut.Tag.Type)
					assert.NotZero(ut.Tag.CreatedAt)
					assert.NotZero(ut.Tag.UpdatedAt)
				}
			}
		})

		t.Run("notfound", func(t *testing.T) {
			t.Parallel()

			_, err := GetUserTag(user.GetUID(), uuid.Nil)
			assert.Error(err)
		})
	})

	// GetUserIDsByTag
	t.Run("TestGetUserIDsByTag", func(t *testing.T) {
		t.Parallel()

		s := utils.RandAlphabetAndNumberString(20)
		tag := mustMakeTag(t, s)
		for i := 0; i < 10; i++ {
			user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
			require.NoError(AddUserTag(user.GetUID(), tag.ID))
		}

		t.Run("found", func(t *testing.T) {
			t.Parallel()

			ids, err := GetUserIDsByTag(s)
			if assert.NoError(err) {
				assert.Len(ids, 10)
			}
		})

		t.Run("notfound", func(t *testing.T) {
			t.Parallel()

			ids, err := GetUserIDsByTag(utils.RandAlphabetAndNumberString(20))
			if assert.NoError(err) {
				assert.Len(ids, 0)
			}
		})
	})

	// GetUsersByTag
	t.Run("TestGetUsersByTag", func(t *testing.T) {
		t.Parallel()

		s := utils.RandAlphabetAndNumberString(20)
		tag := mustMakeTag(t, s)
		for i := 0; i < 10; i++ {
			user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
			require.NoError(AddUserTag(user.GetUID(), tag.ID))
		}

		t.Run("found", func(t *testing.T) {
			t.Parallel()

			ids, err := GetUsersByTag(s)
			if assert.NoError(err) {
				assert.Len(ids, 10)
			}
		})

		t.Run("notfound", func(t *testing.T) {
			t.Parallel()

			ids, err := GetUsersByTag(utils.RandAlphabetAndNumberString(20))
			if assert.NoError(err) {
				assert.Len(ids, 0)
			}
		})
	})

	// GetUserIDsByTagID
	t.Run("TestGetUserIDsByTagID", func(t *testing.T) {
		t.Parallel()

		tag := mustMakeTag(t, utils.RandAlphabetAndNumberString(20))
		for i := 0; i < 10; i++ {
			user := mustMakeUser(t, utils.RandAlphabetAndNumberString(20))
			require.NoError(AddUserTag(user.GetUID(), tag.ID))
		}

		t.Run("found", func(t *testing.T) {
			t.Parallel()

			ids, err := GetUserIDsByTagID(tag.ID)
			if assert.NoError(err) {
				assert.Len(ids, 10)
			}
		})

		t.Run("notfound", func(t *testing.T) {
			t.Parallel()

			ids, err := GetUserIDsByTagID(uuid.Nil)
			if assert.NoError(err) {
				assert.Len(ids, 0)
			}
		})
	})

	// CreateTag
	t.Run("TestCreateTag", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name       string
			restricted bool
			tagType    string
		}{
			{"tagA_" + utils.RandAlphabetAndNumberString(20), false, ""},
			{"tagB_" + utils.RandAlphabetAndNumberString(20), true, "aaaa"},
		}

		for _, v := range cases {
			v := v
			t.Run(v.name, func(t *testing.T) {
				t.Parallel()

				tag, err := CreateTag(v.name, v.restricted, v.tagType)
				if assert.NoError(err) {
					assert.NotZero(tag.ID)
					assert.Equal(v.name, tag.Name)
					assert.Equal(v.restricted, tag.Restricted)
					assert.Equal(v.tagType, tag.Type)
					assert.NotZero(tag.CreatedAt)
					assert.NotZero(tag.UpdatedAt)
					count := 0
					db.Model(Tag{}).Where(&Tag{ID: tag.ID}).Count(&count)
					assert.Equal(1, count)
				}
			})
		}
	})

	// ChangeTagType
	t.Run("TestChangeTagType", func(t *testing.T) {
		t.Parallel()

		tag := mustMakeTag(t, utils.RandAlphabetAndNumberString(20))

		err := ChangeTagType(tag.ID, "newType")
		if assert.NoError(err) {
			t, err := GetTagByID(tag.ID)
			require.NoError(err)
			assert.Equal("newType", t.Type)
		}
	})

	// ChangeTagRestrict
	t.Run("TestChangeTagRestrict", func(t *testing.T) {
		t.Parallel()

		tag := mustMakeTag(t, utils.RandAlphabetAndNumberString(20))

		err := ChangeTagRestrict(tag.ID, true)
		if assert.NoError(err) {
			t, err := GetTagByID(tag.ID)
			require.NoError(err)
			assert.True(t.Restricted)
		}

		err = ChangeTagRestrict(tag.ID, false)
		if assert.NoError(err) {
			t, err := GetTagByID(tag.ID)
			require.NoError(err)
			assert.False(t.Restricted)
		}
	})

	// GetTagByID
	t.Run("TestGetTagByID", func(t *testing.T) {
		t.Parallel()

		tag := mustMakeTag(t, utils.RandAlphabetAndNumberString(20))

		r, err := GetTagByID(tag.ID)
		if assert.NoError(err) {
			assert.Equal(tag.Name, r.Name)
		}

		_, err = GetTagByID(uuid.NewV4())
		assert.Error(err)
	})

	// GetTagByName
	t.Run("TestGetTagByName", func(t *testing.T) {
		t.Parallel()

		s := utils.RandAlphabetAndNumberString(20)
		tag := mustMakeTag(t, s)

		r, err := GetTagByName(s)
		if assert.NoError(err) {
			assert.Equal(tag.ID, r.ID)
		}

		_, err = GetTagByName(utils.RandAlphabetAndNumberString(20))
		assert.Error(err)

		_, err = GetTagByName("")
		assert.Error(err)
	})

	// GetOrCreateTagByName
	t.Run("TestGetOrCreateTagByName", func(t *testing.T) {
		t.Parallel()

		s := utils.RandAlphabetAndNumberString(20)
		tag := mustMakeTag(t, s)

		r, err := GetOrCreateTagByName(s)
		if assert.NoError(err) {
			assert.Equal(tag.ID, r.ID)
		}

		b := utils.RandAlphabetAndNumberString(20)
		r, err = GetOrCreateTagByName(b)
		if assert.NoError(err) {
			assert.NotZero(r.ID)
			assert.Equal(b, r.Name)
			assert.False(r.Restricted)
			assert.Empty(r.Type)
			assert.NotZero(r.CreatedAt)
			assert.NotZero(r.UpdatedAt)
		}

		_, err = GetOrCreateTagByName("")
		assert.Error(err)
	})

}

func TestGetAllTags(t *testing.T) {
	assert, require, _, _ := beforeTest(t)

	_, err := CreateTag("tagA", false, "")
	require.NoError(err)
	_, err = CreateTag("tagB", false, "")
	require.NoError(err)
	_, err = CreateTag("tagC", false, "")
	require.NoError(err)
	_, err = CreateTag("tagD", false, "")
	require.NoError(err)

	tags, err := GetAllTags()
	if assert.NoError(err) {
		assert.Len(tags, 4)
	}
}
