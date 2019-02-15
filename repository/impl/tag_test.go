package impl

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils"
	"testing"
)

func TestRepositoryImpl_AddUserTag(t *testing.T) {
	t.Parallel()
	repo, assert, _, user := setupWithUser(t, common)

	tag := mustMakeTag(t, repo, random)
	assert.NoError(repo.AddUserTag(user.ID, tag.ID))
}

func TestRepositoryImpl_ChangeUserTagLock(t *testing.T) {
	t.Parallel()
	repo, assert, require, user := setupWithUser(t, common)

	tag := mustMakeTag(t, repo, random)
	require.NoError(repo.AddUserTag(user.ID, tag.ID))

	if assert.NoError(repo.ChangeUserTagLock(user.ID, tag.ID, true)) {
		tag, err := repo.GetUserTag(user.ID, tag.ID)
		require.NoError(err)
		assert.True(tag.IsLocked)
	}

	if assert.NoError(repo.ChangeUserTagLock(user.ID, tag.ID, false)) {
		tag, err := repo.GetUserTag(user.ID, tag.ID)
		require.NoError(err)
		assert.False(tag.IsLocked)
	}
}

func TestRepositoryImpl_DeleteUserTag(t *testing.T) {
	t.Parallel()
	repo, assert, require, user := setupWithUser(t, common)

	tag := mustMakeTag(t, repo, random)
	require.NoError(repo.AddUserTag(user.ID, tag.ID))
	tag2 := mustMakeTag(t, repo, random)
	require.NoError(repo.AddUserTag(user.ID, tag2.ID))

	if assert.NoError(repo.DeleteUserTag(user.ID, tag.ID)) {
		_, err := repo.GetUserTag(user.ID, tag.ID)
		assert.Error(err)
	}

	_, err := repo.GetUserTag(user.ID, tag2.ID)
	assert.NoError(err)
}

func TestRepositoryImpl_GetUserTagsByUserID(t *testing.T) {
	t.Parallel()
	repo, _, require, user := setupWithUser(t, common)

	var createdTags []string
	for i := 0; i < 10; i++ {
		tag := mustMakeTag(t, repo, random)
		require.NoError(repo.AddUserTag(user.ID, tag.ID))
		createdTags = append(createdTags, tag.Name)
	}

	t.Run("has", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		tags, err := repo.GetUserTagsByUserID(user.ID)
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
		assert := assert.New(t)

		tags, err := repo.GetUserTagsByUserID(uuid.Nil)
		if assert.NoError(err) {
			assert.Empty(tags)
		}
	})
}

func TestRepositoryImpl_GetUserTag(t *testing.T) {
	t.Parallel()
	repo, _, require, user := setupWithUser(t, common)

	tag := mustMakeTag(t, repo, random)
	require.NoError(repo.AddUserTag(user.ID, tag.ID))

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ut, err := repo.GetUserTag(user.ID, tag.ID)
		if assert.NoError(err) {
			assert.Equal(user.ID, ut.UserID)
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
		assert := assert.New(t)

		_, err := repo.GetUserTag(user.ID, uuid.Nil)
		assert.Error(err)
	})
}

func TestRepositoryImpl_GetUserIDsByTag(t *testing.T) {
	t.Parallel()
	repo, _, require := setup(t, common)

	s := utils.RandAlphabetAndNumberString(20)
	tag := mustMakeTag(t, repo, s)
	for i := 0; i < 10; i++ {
		user := mustMakeUser(t, repo, random)
		require.NoError(repo.AddUserTag(user.ID, tag.ID))
	}

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetUserIDsByTag(s)
		if assert.NoError(err) {
			assert.Len(ids, 10)
		}
	})

	t.Run("notfound", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetUserIDsByTag(utils.RandAlphabetAndNumberString(20))
		if assert.NoError(err) {
			assert.Len(ids, 0)
		}
	})
}

func TestRepositoryImpl_GetUsersByTag(t *testing.T) {
	t.Parallel()
	repo, _, require := setup(t, common)

	s := utils.RandAlphabetAndNumberString(20)
	tag := mustMakeTag(t, repo, s)
	for i := 0; i < 10; i++ {
		user := mustMakeUser(t, repo, random)
		require.NoError(repo.AddUserTag(user.ID, tag.ID))
	}

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetUsersByTag(s)
		if assert.NoError(err) {
			assert.Len(ids, 10)
		}
	})

	t.Run("notfound", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetUsersByTag(utils.RandAlphabetAndNumberString(20))
		if assert.NoError(err) {
			assert.Len(ids, 0)
		}
	})
}

func TestRepositoryImpl_GetUserIDsByTagID(t *testing.T) {
	t.Parallel()
	repo, _, require := setup(t, common)

	tag := mustMakeTag(t, repo, random)
	for i := 0; i < 10; i++ {
		user := mustMakeUser(t, repo, random)
		require.NoError(repo.AddUserTag(user.ID, tag.ID))
	}

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetUserIDsByTagID(tag.ID)
		if assert.NoError(err) {
			assert.Len(ids, 10)
		}
	})

	t.Run("notfound", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetUserIDsByTagID(uuid.Nil)
		if assert.NoError(err) {
			assert.Len(ids, 0)
		}
	})
}

func TestRepositoryImpl_CreateTag(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common)

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
			assert := assert.New(t)

			tag, err := repo.CreateTag(v.name, v.restricted, v.tagType)
			if assert.NoError(err) {
				assert.NotZero(tag.ID)
				assert.Equal(v.name, tag.Name)
				assert.Equal(v.restricted, tag.Restricted)
				assert.Equal(v.tagType, tag.Type)
				assert.NotZero(tag.CreatedAt)
				assert.NotZero(tag.UpdatedAt)
				assert.Equal(1, count(t, getDB(repo).Model(model.Tag{}).Where(&model.Tag{ID: tag.ID})))
			}
		})
	}
}

func TestRepositoryImpl_ChangeTagType(t *testing.T) {
	t.Parallel()
	repo, assert, require := setup(t, common)

	tag := mustMakeTag(t, repo, random)

	err := repo.ChangeTagType(tag.ID, "newType")
	if assert.NoError(err) {
		t, err := repo.GetTagByID(tag.ID)
		require.NoError(err)
		assert.Equal("newType", t.Type)
	}
}

func TestRepositoryImpl_ChangeTagRestrict(t *testing.T) {
	t.Parallel()
	repo, assert, require := setup(t, common)

	tag := mustMakeTag(t, repo, random)

	err := repo.ChangeTagRestrict(tag.ID, true)
	if assert.NoError(err) {
		t, err := repo.GetTagByID(tag.ID)
		require.NoError(err)
		assert.True(t.Restricted)
	}

	err = repo.ChangeTagRestrict(tag.ID, false)
	if assert.NoError(err) {
		t, err := repo.GetTagByID(tag.ID)
		require.NoError(err)
		assert.False(t.Restricted)
	}
}

func TestRepositoryImpl_GetTagByID(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	tag := mustMakeTag(t, repo, random)

	r, err := repo.GetTagByID(tag.ID)
	if assert.NoError(err) {
		assert.Equal(tag.Name, r.Name)
	}

	_, err = repo.GetTagByID(uuid.NewV4())
	assert.Error(err)
}

func TestRepositoryImpl_GetTagByName(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	s := utils.RandAlphabetAndNumberString(20)
	tag := mustMakeTag(t, repo, s)

	r, err := repo.GetTagByName(s)
	if assert.NoError(err) {
		assert.Equal(tag.ID, r.ID)
	}

	_, err = repo.GetTagByName(utils.RandAlphabetAndNumberString(20))
	assert.Error(err)

	_, err = repo.GetTagByName("")
	assert.Error(err)
}

func TestRepositoryImpl_GetOrCreateTagByName(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	s := utils.RandAlphabetAndNumberString(20)
	tag := mustMakeTag(t, repo, s)

	r, err := repo.GetOrCreateTagByName(s)
	if assert.NoError(err) {
		assert.Equal(tag.ID, r.ID)
	}

	b := utils.RandAlphabetAndNumberString(20)
	r, err = repo.GetOrCreateTagByName(b)
	if assert.NoError(err) {
		assert.NotZero(r.ID)
		assert.Equal(b, r.Name)
		assert.False(r.Restricted)
		assert.Empty(r.Type)
		assert.NotZero(r.CreatedAt)
		assert.NotZero(r.UpdatedAt)
	}

	_, err = repo.GetOrCreateTagByName("")
	assert.Error(err)
}

func TestRepositoryImpl_GetAllTags(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, ex1)

	for i := 0; i < 4; i++ {
		mustMakeTag(t, repo, random)
	}

	tags, err := repo.GetAllTags()
	if assert.NoError(err) {
		assert.Len(tags, 4)
	}
}
