package gorm

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	random2 "github.com/traPtitech/traQ/utils/random"
)

func TestRepositoryImpl_AddUserTag(t *testing.T) {
	t.Parallel()
	repo, assert, _, user := setupWithUser(t, common2, false)

	tag := mustMakeTag(t, repo, rand)
	assert.NoError(repo.AddUserTag(user.GetID(), tag.ID))
	assert.Error(repo.AddUserTag(user.GetID(), tag.ID))
	assert.Error(repo.AddUserTag(user.GetID(), uuid.Nil))
}

func TestRepositoryImpl_ChangeUserTagLock(t *testing.T) {
	t.Parallel()
	repo, assert, require, user := setupWithUser(t, common2, false)

	tag := mustMakeTag(t, repo, rand)
	mustAddTagToUser(t, repo, user.GetID(), tag.ID)

	if assert.NoError(repo.ChangeUserTagLock(user.GetID(), tag.ID, true)) {
		tag, err := repo.GetUserTag(user.GetID(), tag.ID)
		require.NoError(err)
		assert.True(tag.GetIsLocked())
	}

	if assert.NoError(repo.ChangeUserTagLock(user.GetID(), tag.ID, false)) {
		tag, err := repo.GetUserTag(user.GetID(), tag.ID)
		require.NoError(err)
		assert.False(tag.GetIsLocked())
	}

	assert.Error(repo.ChangeUserTagLock(uuid.Nil, tag.ID, true))
}

func TestRepositoryImpl_DeleteUserTag(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common2, false)

	tag := mustMakeTag(t, repo, rand)
	mustAddTagToUser(t, repo, user.GetID(), tag.ID)
	tag2 := mustMakeTag(t, repo, rand)
	mustAddTagToUser(t, repo, user.GetID(), tag2.ID)

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		if assert.NoError(repo.DeleteUserTag(user.GetID(), tag.ID)) {
			_, err := repo.GetUserTag(user.GetID(), tag.ID)
			assert.Error(err)
		}

		_, err := repo.GetUserTag(user.GetID(), tag2.ID)
		assert.NoError(err)
	})

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, repo.DeleteUserTag(uuid.Nil, tag.ID))
	})
}

func TestRepositoryImpl_GetUserTagsByUserID(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common2, false)

	var createdTags []string
	for range 10 {
		tag := mustMakeTag(t, repo, rand)
		mustAddTagToUser(t, repo, user.GetID(), tag.ID)
		createdTags = append(createdTags, tag.Name)
	}

	t.Run("has", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		tags, err := repo.GetUserTagsByUserID(user.GetID())
		if assert.NoError(err) {
			temp := make([]string, len(tags))
			for i, v := range tags {
				temp[i] = v.GetTag()
			}
			assert.ElementsMatch(temp, createdTags)
		}
	})

	t.Run("has no", func(t *testing.T) {
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
	repo, _, _, user := setupWithUser(t, common2, false)

	tag := mustMakeTag(t, repo, rand)
	mustAddTagToUser(t, repo, user.GetID(), tag.ID)

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ut, err := repo.GetUserTag(user.GetID(), tag.ID)
		if assert.NoError(err) {
			assert.Equal(user.GetID(), ut.GetUserID())
			assert.Equal(tag.ID, ut.GetTagID())
			assert.False(ut.GetIsLocked())
			assert.NotZero(ut.GetCreatedAt())
			assert.NotZero(ut.GetUpdatedAt())
			assert.NotZero(ut.GetTag())
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		_, err := repo.GetUserTag(user.GetID(), uuid.Nil)
		assert.Error(err)
	})
}

func TestRepositoryImpl_GetUserIDsByTagID(t *testing.T) {
	t.Parallel()
	repo, _, _ := setup(t, common2)

	tag := mustMakeTag(t, repo, rand)
	for range 10 {
		mustAddTagToUser(t, repo, mustMakeUser(t, repo, rand, false).GetID(), tag.ID)
	}

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetUserIDsByTagID(tag.ID)
		if assert.NoError(err) {
			assert.Len(ids, 10)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		ids, err := repo.GetUserIDsByTagID(uuid.Nil)
		if assert.NoError(err) {
			assert.Len(ids, 0)
		}
	})
}

func TestRepositoryImpl_GetTagByID(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common2)

	tag := mustMakeTag(t, repo, rand)

	r, err := repo.GetTagByID(tag.ID)
	if assert.NoError(err) {
		assert.Equal(tag.Name, r.Name)
	}

	_, err = repo.GetTagByID(uuid.Must(uuid.NewV4()))
	assert.Error(err)

	_, err = repo.GetTagByID(uuid.Must(uuid.NewV7()))
	assert.Error(err)

	_, err = repo.GetTagByID(uuid.Nil)
	assert.Error(err)
}

func TestRepositoryImpl_GetOrCreateTagByName(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common2)

	s := random2.AlphaNumeric(20)
	tag := mustMakeTag(t, repo, s)

	r, err := repo.GetOrCreateTag(s)
	if assert.NoError(err) {
		assert.Equal(tag.ID, r.ID)
	}

	b := random2.AlphaNumeric(20)
	r, err = repo.GetOrCreateTag(b)
	if assert.NoError(err) {
		assert.NotZero(r.ID)
		assert.Equal(b, r.Name)
		assert.NotZero(r.CreatedAt)
		assert.NotZero(r.UpdatedAt)
	}

	_, err = repo.GetOrCreateTag("")
	assert.Error(err)

	_, err = repo.GetOrCreateTag(random2.AlphaNumeric(31))
	assert.Error(err)
}
