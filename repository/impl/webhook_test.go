package impl

import (
	"database/sql"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"strings"
	"testing"
)

func TestRepositoryImpl_CreateWebhook(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common)

	t.Run("Invalid name", func(t *testing.T) {
		t.Parallel()
		_, err := repo.CreateWebhook("", "", channel.ID, user.ID, "")
		assert.Error(t, err)
		_, err = repo.CreateWebhook(strings.Repeat("a", 40), "", channel.ID, user.ID, "")
		assert.Error(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)

		wb, err := repo.CreateWebhook("test", "aaa", channel.ID, user.ID, "test")
		if assert.NoError(err) {
			assert.Equal("test", wb.GetName())
			assert.Equal("aaa", wb.GetDescription())
			assert.Equal(channel.ID, wb.GetChannelID())
			assert.Equal(user.ID, wb.GetCreatorID())
			assert.Equal("test", wb.GetSecret())

			u, err := repo.GetUser(wb.GetBotUserID())
			if assert.NoError(err) {
				assert.True(u.Bot)
				assert.Equal(role.Bot.ID(), u.Role)
				assert.Equal(model.UserAccountStatusActive, u.Status)
			}
		}
	})
}

func TestRepositoryImpl_UpdateWebhook(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common)

	t.Run("Nil id", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, repo.UpdateWebhook(uuid.Nil, repository.UpdateWebhookArgs{}), repository.ErrNilID.Error())
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, repo.UpdateWebhook(uuid.Must(uuid.NewV4()), repository.UpdateWebhookArgs{}), repository.ErrNotFound.Error())
	})

	t.Run("Invalid name", func(t *testing.T) {
		t.Parallel()
		wb := mustMakeWebhook(t, repo, random, channel.ID, user.ID, "test")
		err := repo.UpdateWebhook(wb.GetID(), repository.UpdateWebhookArgs{
			Name: sql.NullString{
				Valid:  true,
				String: strings.Repeat("a", 40),
			},
		})
		assert.Error(t, err)
	})

	t.Run("No changes", func(t *testing.T) {
		t.Parallel()
		wb := mustMakeWebhook(t, repo, random, channel.ID, user.ID, "test")
		err := repo.UpdateWebhook(wb.GetID(), repository.UpdateWebhookArgs{})
		assert.NoError(t, err)
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		wb := mustMakeWebhook(t, repo, random, channel.ID, user.ID, "test")
		ch := mustMakeChannel(t, repo, random)
		assert, require := assertAndRequire(t)

		err := repo.UpdateWebhook(wb.GetID(), repository.UpdateWebhookArgs{
			Description: sql.NullString{
				Valid:  true,
				String: "new description",
			},
			Name: sql.NullString{
				Valid:  true,
				String: "new name",
			},
			Secret: sql.NullString{
				Valid:  true,
				String: "new secret",
			},
			ChannelID: uuid.NullUUID{
				Valid: true,
				UUID:  ch.ID,
			},
		})
		if assert.NoError(err) {
			wb, err := repo.GetWebhook(wb.GetID())
			require.NoError(err)
			assert.Equal("new name", wb.GetName())
			assert.Equal("new description", wb.GetDescription())
			assert.Equal("new secret", wb.GetSecret())
			assert.Equal(ch.ID, wb.GetChannelID())
		}
	})
}

func TestRepositoryImpl_DeleteWebhook(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common)

	t.Run("Nil id", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, repo.DeleteWebhook(uuid.Nil), repository.ErrNilID.Error())
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		assert.EqualError(t, repo.DeleteWebhook(uuid.Must(uuid.NewV4())), repository.ErrNotFound.Error())
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)
		wb := mustMakeWebhook(t, repo, random, channel.ID, user.ID, "test")

		err := repo.DeleteWebhook(wb.GetID())
		if assert.NoError(err) {
			_, err := repo.GetWebhook(wb.GetID())
			assert.EqualError(err, repository.ErrNotFound.Error())
		}
	})
}

func TestRepositoryImpl_GetWebhook(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common)

	t.Run("Nil id", func(t *testing.T) {
		t.Parallel()
		_, err := repo.GetWebhook(uuid.Nil)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		_, err := repo.GetWebhook(uuid.Must(uuid.NewV4()))
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)
		wb := mustMakeWebhook(t, repo, random, channel.ID, user.ID, "test")

		w, err := repo.GetWebhook(wb.GetID())
		if assert.NoError(err) {
			assert.Equal(wb.GetID(), w.GetID())
			assert.Equal(wb.GetName(), w.GetName())
			assert.Equal(wb.GetChannelID(), w.GetChannelID())
			assert.Equal(wb.GetSecret(), w.GetSecret())
			assert.Equal(wb.GetDescription(), w.GetDescription())
			assert.Equal(wb.GetCreatorID(), w.GetCreatorID())
			assert.Equal(wb.GetBotUserID(), w.GetBotUserID())
		}
	})
}

func TestRepositoryImpl_GetWebhookByBotUserId(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common)

	t.Run("Nil id", func(t *testing.T) {
		t.Parallel()
		_, err := repo.GetWebhookByBotUserId(uuid.Nil)
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Not found", func(t *testing.T) {
		t.Parallel()
		_, err := repo.GetWebhookByBotUserId(uuid.Must(uuid.NewV4()))
		assert.EqualError(t, err, repository.ErrNotFound.Error())
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		assert, _ := assertAndRequire(t)
		wb := mustMakeWebhook(t, repo, random, channel.ID, user.ID, "test")

		w, err := repo.GetWebhookByBotUserId(wb.GetBotUserID())
		if assert.NoError(err) {
			assert.Equal(wb.GetID(), w.GetID())
			assert.Equal(wb.GetName(), w.GetName())
			assert.Equal(wb.GetChannelID(), w.GetChannelID())
			assert.Equal(wb.GetSecret(), w.GetSecret())
			assert.Equal(wb.GetDescription(), w.GetDescription())
			assert.Equal(wb.GetCreatorID(), w.GetCreatorID())
			assert.Equal(wb.GetBotUserID(), w.GetBotUserID())
		}
	})
}

func TestRepositoryImpl_GetAllWebhooks(t *testing.T) {
	t.Parallel()
	repo, assert, _, user, channel := setupWithUserAndChannel(t, ex3)

	n := 10
	for i := 0; i < n; i++ {
		mustMakeWebhook(t, repo, random, channel.ID, user.ID, "test")
	}

	arr, err := repo.GetAllWebhooks()
	if assert.NoError(err) {
		assert.Len(arr, n)
	}
}

func TestRepositoryImpl_GetWebhooksByCreator(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common)

	n := 10
	for i := 0; i < n; i++ {
		mustMakeWebhook(t, repo, random, channel.ID, user.ID, "test")
	}
	user2 := mustMakeUser(t, repo, random)
	mustMakeWebhook(t, repo, random, channel.ID, user2.ID, "test")

	t.Run("Nil id", func(t *testing.T) {
		t.Parallel()
		arr, err := repo.GetWebhooksByCreator(uuid.Nil)
		if assert.NoError(t, err) {
			assert.Empty(t, arr)
		}
	})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		arr, err := repo.GetWebhooksByCreator(user.ID)
		if assert.NoError(t, err) {
			assert.Len(t, arr, n)
		}
	})
}
