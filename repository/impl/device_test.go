package impl

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils"
	"testing"
)

func TestRepositoryImpl_RegisterDevice(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	id1 := mustMakeUser(t, repo, random).ID
	id2 := mustMakeUser(t, repo, random).ID
	token1 := utils.RandAlphabetAndNumberString(20)
	token2 := utils.RandAlphabetAndNumberString(20)

	cases := []struct {
		user  uuid.UUID
		token string
		error bool
	}{
		{id1, token1, false},
		{id2, token2, false},
		{id2, token2, false},
		{id1, token2, true},
	}

	for _, v := range cases {
		_, err := repo.RegisterDevice(v.user, v.token)
		if v.error {
			assert.Error(err)
		} else {
			assert.NoError(err)
		}
	}

	assert.EqualValues(2, count(t, getDB(repo).Model(model.Device{}).Where("user_id IN (?, ?)", id1, id2)))
}

func TestRepositoryImpl_UnregisterDevice(t *testing.T) {
	t.Parallel()
	repo, assert, require := setup(t, common)

	id1 := mustMakeUser(t, repo, random).ID
	id2 := mustMakeUser(t, repo, random).ID
	token1 := utils.RandAlphabetAndNumberString(20)
	token2 := utils.RandAlphabetAndNumberString(20)
	token3 := utils.RandAlphabetAndNumberString(20)

	_, err := repo.RegisterDevice(id1, token1)
	require.NoError(err)
	_, err = repo.RegisterDevice(id2, token2)
	require.NoError(err)
	_, err = repo.RegisterDevice(id1, token3)
	require.NoError(err)

	cases := []struct {
		token  string
		expect int
	}{
		{token2, 2},
		{"", 2},
		{token3, 1},
	}
	for _, v := range cases {
		assert.NoError(repo.UnregisterDevice(v.token))
		assert.EqualValues(v.expect, count(t, getDB(repo).Model(model.Device{}).Where("user_id IN (?, ?)", id1, id2)))
	}
}

func TestRepositoryImpl_GetDevicesByUserID(t *testing.T) {
	t.Parallel()
	repo, _, require := setup(t, common)

	id1 := mustMakeUser(t, repo, random).ID
	id2 := mustMakeUser(t, repo, random).ID
	token1 := utils.RandAlphabetAndNumberString(20)
	token2 := utils.RandAlphabetAndNumberString(20)
	token3 := utils.RandAlphabetAndNumberString(20)

	_, err := repo.RegisterDevice(id1, token1)
	require.NoError(err)
	_, err = repo.RegisterDevice(id2, token2)
	require.NoError(err)
	_, err = repo.RegisterDevice(id1, token3)
	require.NoError(err)

	cases := []struct {
		name   string
		user   uuid.UUID
		expect int
	}{
		{"id1", id1, 2},
		{"id2", id2, 1},
		{"nil id", uuid.Nil, 0},
	}

	for _, v := range cases {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			devs, err := repo.GetDevicesByUserID(v.user)
			if assert.NoError(t, err) {
				assert.Len(t, devs, v.expect)
			}
		})
	}
}

func TestRepositoryImpl_GetDeviceTokensByUserID(t *testing.T) {
	t.Parallel()
	repo, _, require := setup(t, common)

	id1 := mustMakeUser(t, repo, random).ID
	id2 := mustMakeUser(t, repo, random).ID
	token1 := utils.RandAlphabetAndNumberString(20)
	token2 := utils.RandAlphabetAndNumberString(20)
	token3 := utils.RandAlphabetAndNumberString(20)

	_, err := repo.RegisterDevice(id1, token1)
	require.NoError(err)
	_, err = repo.RegisterDevice(id2, token2)
	require.NoError(err)
	_, err = repo.RegisterDevice(id1, token3)
	require.NoError(err)

	cases := []struct {
		name   string
		user   uuid.UUID
		expect int
	}{
		{"id1", id1, 2},
		{"id2", id2, 1},
		{"nil id", uuid.Nil, 0},
	}

	for _, v := range cases {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			assert := assert.New(t)
			devs, err := repo.GetDeviceTokensByUserID(v.user)
			if assert.NoError(err) {
				assert.Len(devs, v.expect)
			}
		})
	}
}

func TestRepositoryImpl_GetAllDevices(t *testing.T) {
	t.Parallel()
	repo, _, require := setup(t, ex1)

	id1 := mustMakeUser(t, repo, random).ID
	id2 := mustMakeUser(t, repo, random).ID
	token1 := utils.RandAlphabetAndNumberString(20)
	token2 := utils.RandAlphabetAndNumberString(20)
	token3 := utils.RandAlphabetAndNumberString(20)

	_, err := repo.RegisterDevice(id1, token1)
	require.NoError(err)
	_, err = repo.RegisterDevice(id2, token2)
	require.NoError(err)
	_, err = repo.RegisterDevice(id1, token3)
	require.NoError(err)

	// GetAllDevices
	t.Run("TestGetAllDevices", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		devs, err := repo.GetAllDevices()
		if assert.NoError(err) {
			assert.Len(devs, 3)
		}
	})

	// GetAllDeviceTokens
	t.Run("GetAllDeviceTokens", func(t *testing.T) {
		t.Parallel()
		assert := assert.New(t)

		devs, err := repo.GetAllDeviceTokens()
		if assert.NoError(err) {
			assert.Len(devs, 3)
		}
	})
}
