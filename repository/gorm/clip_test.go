package gorm

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
	random2 "github.com/traPtitech/traQ/utils/random"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

func TestRepositoryImpl_CreateClipFolder(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common3)
	t.Run("nil user id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		_, err := repo.CreateClipFolder(uuid.Nil, random2.AlphaNumeric(20), random2.AlphaNumeric(100))

		assert.Error(err)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		name := random2.AlphaNumeric(20)
		description := random2.AlphaNumeric(100)
		cf, err := repo.CreateClipFolder(user.GetID(), name, description)

		if assert.NoError(err) {
			assert.NotEmpty(cf.ID)
			assert.NotEmpty(cf.CreatedAt)
			assert.Equal(name, cf.Name)
			assert.Equal(description, cf.Description)
			assert.Equal(user.GetID(), cf.OwnerID)
		}

	})
}

func TestRepositoryImpl_UpdateClipFolder(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common3)

	clipFolder := mustMakeClipFolder(t, repo, user.GetID(), rand, rand)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.EqualError(repo.UpdateClipFolder(uuid.Nil, optional.Of[string]{}, optional.Of[string]{}), repository.ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.EqualError(repo.UpdateClipFolder(uuid.Must(uuid.NewV7()), optional.Of[string]{}, optional.Of[string]{}), repository.ErrNotFound.Error())
	})

	t.Run("no change", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.NoError(repo.UpdateClipFolder(clipFolder.ID, optional.Of[string]{}, optional.Of[string]{}))
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert, require := assertAndRequire(t)

		newName := random2.AlphaNumeric(20)
		newDescription := random2.AlphaNumeric(100)

		if assert.NoError(repo.UpdateClipFolder(clipFolder.ID, optional.From(newName), optional.From(newDescription))) {
			newClipFolder, err := repo.GetClipFolder(clipFolder.ID)
			require.NoError(err)
			assert.Equal(newDescription, newClipFolder.Description)
			assert.Equal(newName, newClipFolder.Name)
		}
	})
}

func TestRepositoryImpl_DeleteClipFolder(t *testing.T) {
	t.Parallel()
	repo, _, _, user := setupWithUser(t, common3)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.EqualError(repo.DeleteClipFolder(uuid.Nil), repository.ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.EqualError(repo.DeleteClipFolder(uuid.Must(uuid.NewV7())), repository.ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		clipFolder := mustMakeClipFolder(t, repo, user.GetID(), rand, rand)

		if assert.NoError(repo.DeleteClipFolder(clipFolder.ID)) {
			_, err := repo.GetClipFolder(clipFolder.ID)
			assert.EqualError(err, repository.ErrNotFound.Error())
		}
	})

}

func TestRepositoryImpl_DeleteClipFolderMessage(t *testing.T) {
	t.Parallel()
	repo, _, _, user, channel := setupWithUserAndChannel(t, common3)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.EqualError(repo.DeleteClipFolderMessage(uuid.Nil, uuid.Nil), repository.ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		assert.EqualError(repo.DeleteClipFolderMessage(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())), repository.ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		clipFolder := mustMakeClipFolder(t, repo, user.GetID(), rand, rand)
		message := mustMakeMessage(t, repo, user.GetID(), channel.ID)
		clipFolderMessage := mustMakeClipFolderMessage(t, repo, clipFolder.ID, message.ID)

		if assert.NoError(repo.DeleteClipFolderMessage(clipFolderMessage.FolderID, clipFolderMessage.MessageID)) {
			messages, _, err := repo.GetClipFolderMessages(clipFolderMessage.FolderID, repository.ClipFolderMessageQuery{})
			assert.Equal([]*model.ClipFolderMessage{}, messages)
			assert.NoError(err)
		}

	})
}

func TestRepositoryImpl_AddClipFolderMessage(t *testing.T) {
	repo, _, _, user, channel := setupWithUserAndChannel(t, common3)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		_, err := repo.AddClipFolderMessage(uuid.Nil, uuid.Nil)
		assert.EqualError(err, repository.ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		_, err := repo.AddClipFolderMessage(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
		assert.EqualError(err, repository.ErrNotFound.Error())
	})

	t.Run("already exist", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		clipFolder := mustMakeClipFolder(t, repo, user.GetID(), rand, rand)
		message := mustMakeMessage(t, repo, user.GetID(), channel.ID)
		clipFolderMessage := mustMakeClipFolderMessage(t, repo, clipFolder.ID, message.ID)

		_, err := repo.AddClipFolderMessage(clipFolderMessage.FolderID, clipFolderMessage.MessageID)
		assert.EqualError(err, repository.ErrAlreadyExists.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		clipFolder := mustMakeClipFolder(t, repo, user.GetID(), rand, rand)
		message := mustMakeMessage(t, repo, user.GetID(), channel.ID)

		cfm, err := repo.AddClipFolderMessage(clipFolder.ID, message.ID)

		if assert.NoError(err) {
			assert.Equal(message.ID, cfm.MessageID)
			assert.Equal(clipFolder.ID, cfm.FolderID)
			assert.NotEmpty(cfm.CreatedAt)
			assert.Equal(message.ID, cfm.MessageID)
		}

	})
}

func TestRepositoryImpl_GetClipFoldersByUserID(t *testing.T) {
	repo, _, _, user := setupWithUser(t, common3)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		_, err := repo.GetClipFoldersByUserID(uuid.Nil)
		assert.EqualError(err, repository.ErrNilID.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		otherUser := mustMakeUser(t, repo, rand)

		n := 10
		for i := 0; i < 10; i++ {
			mustMakeClipFolder(t, repo, user.GetID(), rand, rand)
		}
		mustMakeClipFolder(t, repo, otherUser.GetID(), rand, rand)

		clipFolders, err := repo.GetClipFoldersByUserID(user.GetID())

		if assert.NoError(err) {
			assert.Len(clipFolders, n)
			for _, clipFolder := range clipFolders {
				assert.Equal(user.GetID(), clipFolder.OwnerID)
			}
		}
	})
}

func TestRepositoryImpl_GetClipFolder(t *testing.T) {
	repo, _, _, user := setupWithUser(t, common3)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		_, err := repo.GetClipFolder(uuid.Nil)
		assert.EqualError(err, repository.ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		_, err := repo.GetClipFolder(uuid.Must(uuid.NewV7()))
		assert.EqualError(err, repository.ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		createdClipFolder := mustMakeClipFolder(t, repo, user.GetID(), rand, rand)
		clipFolder, err := repo.GetClipFolder(createdClipFolder.ID)

		if assert.NoError(err) {
			assert.Equal(createdClipFolder.Description, clipFolder.Description)
			assert.Equal(createdClipFolder.ID, clipFolder.ID)
			assert.Equal(createdClipFolder.Name, clipFolder.Name)
			assert.Equal(createdClipFolder.OwnerID, clipFolder.OwnerID)
		}

	})
}

func TestRepositoryImpl_GetClipFolderMessages(t *testing.T) {
	repo, _, _, user, channel := setupWithUserAndChannel(t, common3)

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		_, _, err := repo.GetClipFolderMessages(uuid.Nil, repository.ClipFolderMessageQuery{})
		assert.EqualError(err, repository.ErrNilID.Error())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)
		_, _, err := repo.GetClipFolderMessages(uuid.Must(uuid.NewV4()), repository.ClipFolderMessageQuery{})
		assert.EqualError(err, repository.ErrNotFound.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		createdClipFolder := mustMakeClipFolder(t, repo, user.GetID(), rand, rand)
		createdClipFolderMessages := make([]*model.ClipFolderMessage, 10)
		createdMessages := make([]*model.Message, 10)

		n := 10
		for i := 0; i < 10; i++ {
			createdMessages[i] = mustMakeMessage(t, repo, user.GetID(), channel.ID)
			createdClipFolderMessages[i] = mustMakeClipFolderMessage(t, repo, createdClipFolder.ID, createdMessages[i].ID)
		}

		otherCreatedClipFolder := mustMakeClipFolder(t, repo, user.GetID(), rand, rand)
		otherCreatedMessage := mustMakeMessage(t, repo, user.GetID(), channel.ID)
		mustMakeClipFolderMessage(t, repo, otherCreatedClipFolder.ID, otherCreatedMessage.ID)

		clipFolderMessages, more, err := repo.GetClipFolderMessages(createdClipFolder.ID, repository.ClipFolderMessageQuery{})

		if assert.NoError(err) {
			assert.EqualValues(false, more)
			assert.Len(clipFolderMessages, n)
			for i, clipFolderMessage := range clipFolderMessages {
				assert.Equal(createdClipFolderMessages[n-i-1].FolderID, clipFolderMessage.FolderID)
				assert.Equal(createdClipFolderMessages[n-i-1].MessageID, clipFolderMessage.MessageID)
			}
		}
	})
}
