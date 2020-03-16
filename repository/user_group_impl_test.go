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
	repo, _, _, user := setupWithUser(t, common)

	// Success
	a := utils.RandAlphabetAndNumberString(20)
	if g, err := repo.CreateUserGroup(a, "", "", user.GetID()); assert.NoError(t, err) {
		assert.NotNil(t, g)
	}

	t.Run("duplicate", func(t *testing.T) {
		t.Parallel()

		_, err := repo.CreateUserGroup(a, "", "", user.GetID())
		assert.EqualError(t, err, ErrAlreadyExists.Error())
	})

	t.Run("invalid name", func(t *testing.T) {
		t.Parallel()

		_, err := repo.CreateUserGroup(strings.Repeat("a", 31), "", "", uuid.Nil)
		assert.Error(t, err)
	})

	t.Run("invalid type", func(t *testing.T) {
		t.Parallel()

		_, err := repo.CreateUserGroup(utils.RandAlphabetAndNumberString(20), "", strings.Repeat("a", 31), user.GetID())
		assert.Error(t, err)
	})
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
		g := mustMakeUserGroup(t, repo, random, user.GetID())

		a := utils.RandAlphabetAndNumberString(20)
		if assert.NoError(repo.UpdateUserGroup(g.ID, UpdateUserGroupNameArgs{
			Name:        null.StringFrom(a),
			Description: null.StringFrom(a),
			Type:        null.StringFrom(a),
		})) {
			g, err := repo.GetUserGroup(g.ID)
			require.NoError(err)
			assert.Equal(a, g.Name)
			assert.Equal(a, g.Description)
			assert.Equal(a, g.Type)
		}
	})

	t.Run("no change", func(t *testing.T) {
		t.Parallel()
		g := mustMakeUserGroup(t, repo, random, user.GetID())

		assert.NoError(t, repo.UpdateUserGroup(g.ID, UpdateUserGroupNameArgs{}))
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.UpdateUserGroup(uuid.Must(uuid.NewV4()), UpdateUserGroupNameArgs{}), ErrNotFound.Error())
	})

	t.Run("duplicate", func(t *testing.T) {
		t.Parallel()
		a := utils.RandAlphabetAndNumberString(20)
		mustMakeUserGroup(t, repo, a, user.GetID())
		g := mustMakeUserGroup(t, repo, random, user.GetID())

		assert.EqualError(t, repo.UpdateUserGroup(g.ID, UpdateUserGroupNameArgs{Name: null.StringFrom(a)}), ErrAlreadyExists.Error())
	})

	t.Run("too long name", func(t *testing.T) {
		t.Parallel()
		g := mustMakeUserGroup(t, repo, random, user.GetID())

		assert.Error(t, repo.UpdateUserGroup(g.ID, UpdateUserGroupNameArgs{Name: null.StringFrom(strings.Repeat("a", 31))}))
	})

	t.Run("invalid type", func(t *testing.T) {
		t.Parallel()
		g := mustMakeUserGroup(t, repo, random, user.GetID())

		assert.Error(t, repo.UpdateUserGroup(g.ID, UpdateUserGroupNameArgs{Type: null.StringFrom(strings.Repeat("a", 31))}))
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
		g := mustMakeUserGroup(t, repo, random, user.GetID())

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
		g := mustMakeUserGroup(t, repo, random, user.GetID())

		a, err := repo.GetUserGroup(g.ID)
		if assert.NoError(err) {
			assert.Equal(g.ID, a.ID)
			assert.Equal(g.Name, a.Name)
			assert.Equal(g.Description, a.Description)
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
		g := mustMakeUserGroup(t, repo, random, user.GetID())

		a, err := repo.GetUserGroupByName(g.Name)
		if assert.NoError(err) {
			assert.Equal(g.ID, a.ID)
			assert.Equal(g.Name, a.Name)
			assert.Equal(g.Description, a.Description)
		}
	})
}

func TestRepositoryImpl_GetUserBelongingGroups(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	user2 := mustMakeUser(t, repo, random)
	g1 := mustMakeUserGroup(t, repo, random, user.GetID())
	g2 := mustMakeUserGroup(t, repo, random, user.GetID())
	mustMakeUserGroup(t, repo, random, user.GetID())

	mustAddUserToGroup(t, repo, user.GetID(), g1.ID)
	mustAddUserToGroup(t, repo, user.GetID(), g2.ID)
	mustAddUserToGroup(t, repo, user2.GetID(), g1.ID)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		gs, err := repo.GetUserBelongingGroupIDs(uuid.Nil)
		if assert.NoError(t, err) {
			assert.Empty(t, gs)
		}
	})

	t.Run("success1", func(t *testing.T) {
		t.Parallel()

		gs, err := repo.GetUserBelongingGroupIDs(user.GetID())
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, gs, []uuid.UUID{g1.ID, g2.ID})
		}
	})

	t.Run("success2", func(t *testing.T) {
		t.Parallel()

		gs, err := repo.GetUserBelongingGroupIDs(user2.GetID())
		if assert.NoError(t, err) {
			assert.ElementsMatch(t, gs, []uuid.UUID{g1.ID})
		}
	})
}

func TestRepositoryImpl_GetAllUserGroups(t *testing.T) {
	t.Parallel()
	repo, assert, _, user := setupWithUser(t, ex1)

	mustMakeUserGroup(t, repo, random, user.GetID())
	mustMakeUserGroup(t, repo, random, user.GetID())
	mustMakeUserGroup(t, repo, random, user.GetID())

	gs, err := repo.GetAllUserGroups()
	if assert.NoError(err) {
		assert.Len(gs, 3)
	}
}

func TestRepositoryImpl_AddUserToGroup(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	g := mustMakeUserGroup(t, repo, random, user.GetID())

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.AddUserToGroup(uuid.Nil, g.ID, ""), ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, repo.AddUserToGroup(user.GetID(), g.ID, ""))
		assert.NoError(t, repo.AddUserToGroup(user.GetID(), g.ID, ""))
	})
}

func TestRepositoryImpl_RemoveUserFromGroup(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common)

	g := mustMakeUserGroup(t, repo, random, user.GetID())
	mustAddUserToGroup(t, repo, user.GetID(), g.ID)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.RemoveUserFromGroup(uuid.Nil, g.ID), ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, repo.RemoveUserFromGroup(user.GetID(), g.ID))
		assert.NoError(t, repo.RemoveUserFromGroup(user.GetID(), g.ID))
	})
}
