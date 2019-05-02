package repository

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/utils"
	"gopkg.in/guregu/null.v3"
	"strings"
	"testing"
)

func TestRepositoryImpl_CreateUserGroup(t *testing.T) {
	t.Parallel()
	repo, assert, _, user := setupWithUser(t, common)

	a := utils.RandAlphabetAndNumberString(20)
	if g, err := repo.CreateUserGroup(a, "", "", user.ID); assert.NoError(err) {
		assert.NotNil(g)
	}

	_, err := repo.CreateUserGroup(a, "", "", user.ID)
	assert.EqualError(err, ErrAlreadyExists.Error())

	_, err = repo.CreateUserGroup(strings.Repeat("a", 31), "", "", uuid.Nil)
	assert.Error(err)
}

func TestRepositoryImpl_UpdateUserGroup(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.UpdateUserGroup(uuid.Nil, UpdateUserGroupNameArgs{}), ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, require := assertAndRequire(t)
		g := mustMakeUserGroup(t, repo, random, user.ID)

		a := utils.RandAlphabetAndNumberString(20)
		if assert.NoError(repo.UpdateUserGroup(g.ID, UpdateUserGroupNameArgs{
			Name:        null.StringFrom(a),
			Description: null.StringFrom(a),
			AdminUserID: uuid.NullUUID{
				Valid: true,
				UUID:  user.ID,
			},
		})) {
			g, err := repo.GetUserGroup(g.ID)
			require.NoError(err)
			assert.Equal(a, g.Name)
			assert.Equal(a, g.Description)
			assert.Equal(user.ID, g.AdminUserID)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.UpdateUserGroup(uuid.Must(uuid.NewV4()), UpdateUserGroupNameArgs{}), ErrNotFound.Error())
	})

	t.Run("duplicate", func(t *testing.T) {
		t.Parallel()
		a := utils.RandAlphabetAndNumberString(20)
		mustMakeUserGroup(t, repo, a, user.ID)
		g := mustMakeUserGroup(t, repo, random, user.ID)

		assert.EqualError(t, repo.UpdateUserGroup(g.ID, UpdateUserGroupNameArgs{Name: null.StringFrom(a)}), ErrAlreadyExists.Error())
	})

	t.Run("too long name", func(t *testing.T) {
		t.Parallel()
		g := mustMakeUserGroup(t, repo, random, user.ID)

		assert.Error(t, repo.UpdateUserGroup(g.ID, UpdateUserGroupNameArgs{Name: null.StringFrom(strings.Repeat("a", 31))}))
	})
}

func TestRepositoryImpl_DeleteUserGroup(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, repo.DeleteUserGroup(uuid.Nil))
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.DeleteUserGroup(uuid.Must(uuid.NewV4())), ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		g := mustMakeUserGroup(t, repo, random, user.ID)

		assert.NoError(t, repo.DeleteUserGroup(g.ID))
	})
}

func TestRepositoryImpl_GetUserGroup(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetUserGroup(uuid.Nil)
		assert.EqualError(t, err, ErrNotFound.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetUserGroup(uuid.Must(uuid.NewV4()))
		assert.EqualError(t, err, ErrNotFound.Error())
	})

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		g := mustMakeUserGroup(t, repo, random, user.ID)

		a, err := repo.GetUserGroup(g.ID)
		if assert.NoError(err) {
			assert.Equal(g.ID, a.ID)
			assert.Equal(g.Name, a.Name)
			assert.Equal(g.Description, a.Description)
			assert.Equal(g.AdminUserID, a.AdminUserID)
		}
	})
}

func TestRepositoryImpl_GetUserGroupByName(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetUserGroupByName("")
		assert.EqualError(t, err, ErrNotFound.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetUserGroupByName(utils.RandAlphabetAndNumberString(20))
		assert.EqualError(t, err, ErrNotFound.Error())
	})

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		g := mustMakeUserGroup(t, repo, random, user.ID)

		a, err := repo.GetUserGroupByName(g.Name)
		if assert.NoError(err) {
			assert.Equal(g.ID, a.ID)
			assert.Equal(g.Name, a.Name)
			assert.Equal(g.Description, a.Description)
			assert.Equal(g.AdminUserID, a.AdminUserID)
		}
	})
}

func TestRepositoryImpl_GetUserBelongingGroups(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	user2 := mustMakeUser(t, repo, random)
	g1 := mustMakeUserGroup(t, repo, random, user.ID)
	g2 := mustMakeUserGroup(t, repo, random, user.ID)
	mustMakeUserGroup(t, repo, random, user.ID)

	mustAddUserToGroup(t, repo, user.ID, g1.ID)
	mustAddUserToGroup(t, repo, user.ID, g2.ID)
	mustAddUserToGroup(t, repo, user2.ID, g1.ID)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		gs, err := repo.GetUserBelongingGroupIDs(uuid.Nil)
		if assert.NoError(t, err) {
			assert.Empty(t, gs)
		}
	})

	t.Run("success1", func(t *testing.T) {
		t.Parallel()

		gs, err := repo.GetUserBelongingGroupIDs(user.ID)
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, gs, []uuid.UUID{g1.ID, g2.ID})
		}
	})

	t.Run("success2", func(t *testing.T) {
		t.Parallel()

		gs, err := repo.GetUserBelongingGroupIDs(user2.ID)
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, gs, []uuid.UUID{g1.ID})
		}
	})
}

func TestRepositoryImpl_GetAllUserGroups(t *testing.T) {
	t.Parallel()
	repo, assert, _, user := setupWithUser(t, ex1)

	mustMakeUserGroup(t, repo, random, user.ID)
	mustMakeUserGroup(t, repo, random, user.ID)
	mustMakeUserGroup(t, repo, random, user.ID)

	gs, err := repo.GetAllUserGroups()
	if assert.NoError(err) {
		assert.Len(gs, 3)
	}
}

func TestRepositoryImpl_AddUserToGroup(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	g := mustMakeUserGroup(t, repo, random, user.ID)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.AddUserToGroup(uuid.Nil, g.ID), ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, repo.AddUserToGroup(user.ID, g.ID))
		assert.NoError(t, repo.AddUserToGroup(user.ID, g.ID))
	})
}

func TestRepositoryImpl_RemoveUserFromGroup(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	g := mustMakeUserGroup(t, repo, random, user.ID)
	mustAddUserToGroup(t, repo, user.ID, g.ID)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.RemoveUserFromGroup(uuid.Nil, g.ID), ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, repo.RemoveUserFromGroup(user.ID, g.ID))
		assert.NoError(t, repo.RemoveUserFromGroup(user.ID, g.ID))
	})
}

func TestRepositoryImpl_GetUserGroupMemberIDs(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	user2 := mustMakeUser(t, repo, random)
	g1 := mustMakeUserGroup(t, repo, random, user.ID)
	g2 := mustMakeUserGroup(t, repo, random, user.ID)
	mustMakeUserGroup(t, repo, random, user.ID)

	mustAddUserToGroup(t, repo, user.ID, g1.ID)
	mustAddUserToGroup(t, repo, user.ID, g2.ID)
	mustAddUserToGroup(t, repo, user2.ID, g1.ID)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetUserGroupMemberIDs(uuid.Nil)
		assert.Error(t, err, ErrNotFound.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetUserGroupMemberIDs(uuid.Must(uuid.NewV4()))
		assert.Error(t, err, ErrNotFound.Error())
	})

	t.Run("success1", func(t *testing.T) {
		t.Parallel()

		ids, err := repo.GetUserGroupMemberIDs(g1.ID)
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, ids, []uuid.UUID{user.ID, user2.ID})
		}
	})

	t.Run("success2", func(t *testing.T) {
		t.Parallel()

		ids, err := repo.GetUserGroupMemberIDs(g2.ID)
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, ids, []uuid.UUID{user.ID})
		}
	})
}
