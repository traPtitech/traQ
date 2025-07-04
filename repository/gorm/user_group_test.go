package gorm

import (
	"sync"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
	random2 "github.com/traPtitech/traQ/utils/random"
)

func TestRepositoryImpl_CreateUserGroup(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common3)
	file := mustMakeDummyFile(t, repo)

	// Success
	a := random2.AlphaNumeric(20)
	if g, err := repo.CreateUserGroup(a, "", "", user.GetID(), file.ID); assert.NoError(t, err) {
		assert.NotNil(t, g)
	}

	t.Run("duplicate", func(t *testing.T) {
		t.Parallel()

		_, err := repo.CreateUserGroup(a, "", "", user.GetID(), file.ID)
		assert.EqualError(t, err, repository.ErrAlreadyExists.Error())
	})
}

func TestRepositoryImpl_UpdateUserGroup(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common3)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.UpdateUserGroup(uuid.Nil, repository.UpdateUserGroupArgs{}), repository.ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, require := assertAndRequire(t)
		g := mustMakeUserGroup(t, repo, rand, user.GetID())

		a := random2.AlphaNumeric(20)
		if assert.NoError(repo.UpdateUserGroup(g.ID, repository.UpdateUserGroupArgs{
			Name:        optional.From(a),
			Description: optional.From(a),
			Type:        optional.From(a),
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
		g := mustMakeUserGroup(t, repo, rand, user.GetID())

		assert.NoError(t, repo.UpdateUserGroup(g.ID, repository.UpdateUserGroupArgs{}))
	})

	t.Run("not found(UUIDv4)", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.UpdateUserGroup(uuid.Must(uuid.NewV4()), repository.UpdateUserGroupArgs{}), repository.ErrNotFound.Error())
	})

	t.Run("not found(UUIDv7)", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.UpdateUserGroup(uuid.Must(uuid.NewV7()), repository.UpdateUserGroupArgs{}), repository.ErrNotFound.Error())
	})

	t.Run("duplicate", func(t *testing.T) {
		t.Parallel()
		a := random2.AlphaNumeric(20)
		mustMakeUserGroup(t, repo, a, user.GetID())
		g := mustMakeUserGroup(t, repo, rand, user.GetID())

		assert.EqualError(t, repo.UpdateUserGroup(g.ID, repository.UpdateUserGroupArgs{Name: optional.From(a)}), repository.ErrAlreadyExists.Error())
	})
}

func TestRepositoryImpl_DeleteUserGroup(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common3)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.Error(t, repo.DeleteUserGroup(uuid.Nil))
	})

	t.Run("not found(UUIDv4)", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.DeleteUserGroup(uuid.Must(uuid.NewV4())), repository.ErrNotFound.Error())
	})

	t.Run("not found(UUIDv7)", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.DeleteUserGroup(uuid.Must(uuid.NewV7())), repository.ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		g := mustMakeUserGroup(t, repo, rand, user.GetID())

		assert.NoError(t, repo.DeleteUserGroup(g.ID))
	})
}

func TestRepositoryImpl_GetUserGroup(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common3)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetUserGroup(uuid.Nil)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("not found(UUIDv4)", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetUserGroup(uuid.Must(uuid.NewV4()))
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("not found(UUIDv7)", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetUserGroup(uuid.Must(uuid.NewV7()))
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		g := mustMakeUserGroup(t, repo, rand, user.GetID())

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
	repo, _, _, user := setupWithUser(t, common3)

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetUserGroupByName("")
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		_, err := repo.GetUserGroupByName(random2.AlphaNumeric(20))
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		g := mustMakeUserGroup(t, repo, rand, user.GetID())

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
	repo, _, _, user := setupWithUser(t, common3)

	user2 := mustMakeUser(t, repo, rand)
	g1 := mustMakeUserGroup(t, repo, rand, user.GetID())
	g2 := mustMakeUserGroup(t, repo, rand, user.GetID())
	mustMakeUserGroup(t, repo, rand, user.GetID())

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

	mustMakeUserGroup(t, repo, rand, user.GetID())
	mustMakeUserGroup(t, repo, rand, user.GetID())
	mustMakeUserGroup(t, repo, rand, user.GetID())

	gs, err := repo.GetAllUserGroups()
	if assert.NoError(err) {
		assert.Len(gs, 3)
	}
}

func TestRepositoryImpl_AddUserToGroup(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common3)

	g := mustMakeUserGroup(t, repo, rand, user.GetID())

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.AddUserToGroup(uuid.Nil, g.ID, ""), repository.ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, repo.AddUserToGroup(user.GetID(), g.ID, ""))

		g, err := repo.GetUserGroup(g.ID)
		require.NoError(t, err)
		assert.True(t, g.IsMember(user.GetID()))

		assert.NoError(t, repo.AddUserToGroup(user.GetID(), g.ID, ""))
	})

	t.Run("success concurrently", func(t *testing.T) {
		t.Parallel()
		g := mustMakeUserGroup(t, repo, rand, user.GetID())

		wg := sync.WaitGroup{}
		for range 3 {
			wg.Add(1)
			go func(t *testing.T) {
				defer wg.Done()
				assert.NoError(t, repo.AddUserToGroup(user.GetID(), g.ID, ""))
			}(t)
		}

		wg.Wait()
	})
}

func TestRepositoryImpl_RemoveUserFromGroup(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common3)

	g := mustMakeUserGroup(t, repo, rand, user.GetID())
	mustAddUserToGroup(t, repo, user.GetID(), g.ID)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.RemoveUserFromGroup(uuid.Nil, g.ID), repository.ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, repo.RemoveUserFromGroup(user.GetID(), g.ID))

		g, err := repo.GetUserGroup(g.ID)
		require.NoError(t, err)
		assert.False(t, g.IsMember(user.GetID()))

		assert.NoError(t, repo.RemoveUserFromGroup(user.GetID(), g.ID))
	})
}

func TestRepositoryImpl_AddUserToGroupAdmin(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common3)
	user2 := mustMakeUser(t, repo, rand)

	g := mustMakeUserGroup(t, repo, rand, user.GetID())

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.AddUserToGroupAdmin(uuid.Nil, g.ID), repository.ErrNilID.Error())
	})

	t.Run("not found(UUIDv4)", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.AddUserToGroupAdmin(user2.GetID(), uuid.Must(uuid.NewV4())), repository.ErrNotFound.Error())
	})

	t.Run("not found(UUIDv7)", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.AddUserToGroupAdmin(user2.GetID(), uuid.Must(uuid.NewV7())), repository.ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, repo.AddUserToGroupAdmin(user2.GetID(), g.ID))

		g, err := repo.GetUserGroup(g.ID)
		require.NoError(t, err)
		assert.True(t, g.IsAdmin(user.GetID()))
		assert.True(t, g.IsAdmin(user2.GetID()))

		assert.NoError(t, repo.AddUserToGroupAdmin(user2.GetID(), g.ID))
	})
}

func TestRepositoryImpl_RemoveUserFromGroupAdmin(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common3)
	user2 := mustMakeUser(t, repo, rand)

	g := mustMakeUserGroup(t, repo, rand, user.GetID())
	require.NoError(t, repo.AddUserToGroupAdmin(user2.GetID(), g.ID))

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.RemoveUserFromGroupAdmin(uuid.Nil, g.ID), repository.ErrNilID.Error())
	})

	t.Run("not found(UUIDv4)", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.RemoveUserFromGroupAdmin(user2.GetID(), uuid.Must(uuid.NewV4())), repository.ErrNotFound.Error())
	})

	t.Run("not found(UUIDv7)", func(t *testing.T) {
		t.Parallel()

		assert.EqualError(t, repo.RemoveUserFromGroupAdmin(user2.GetID(), uuid.Must(uuid.NewV7())), repository.ErrNotFound.Error())
	})

	t.Run("cannot remove last admin", func(t *testing.T) {
		t.Parallel()
		g2 := mustMakeUserGroup(t, repo, rand, user.GetID())

		assert.EqualError(t, repo.RemoveUserFromGroupAdmin(user.GetID(), g2.ID), repository.ErrForbidden.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		assert.NoError(t, repo.RemoveUserFromGroupAdmin(user2.GetID(), g.ID))

		g, err := repo.GetUserGroup(g.ID)
		require.NoError(t, err)
		assert.True(t, g.IsAdmin(user.GetID()))
		assert.False(t, g.IsAdmin(user2.GetID()))

		assert.NoError(t, repo.RemoveUserFromGroupAdmin(user2.GetID(), g.ID))
	})
}
