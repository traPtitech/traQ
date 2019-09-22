package repository

import (
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/set"
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
		{uuid.Nil, token2, true},
		{id1, "", true},
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

func TestRepositoryImpl_DeleteDeviceTokens(t *testing.T) {
	t.Parallel()
	repo, assert, require := setup(t, common)

	id1 := mustMakeUser(t, repo, random).ID
	id2 := mustMakeUser(t, repo, random).ID
	token1 := utils.RandAlphabetAndNumberString(20)
	token2 := utils.RandAlphabetAndNumberString(20)
	token3 := utils.RandAlphabetAndNumberString(20)
	token4 := utils.RandAlphabetAndNumberString(20)

	_, err := repo.RegisterDevice(id1, token1)
	require.NoError(err)
	_, err = repo.RegisterDevice(id2, token2)
	require.NoError(err)
	_, err = repo.RegisterDevice(id1, token3)
	require.NoError(err)
	_, err = repo.RegisterDevice(id1, token4)
	require.NoError(err)

	cases := []struct {
		tokens []string
		expect int
	}{
		{[]string{token2}, 3},
		{[]string{}, 3},
		{[]string{token1, token3, ""}, 1},
		{[]string{token4, token2}, 0},
	}
	for _, v := range cases {
		assert.NoError(repo.DeleteDeviceTokens(v.tokens))
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

func TestRepositoryImpl_GetDeviceTokens(t *testing.T) {
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
		users  []uuid.UUID
		expect int
	}{
		{"id1", []uuid.UUID{id1}, 2},
		{"id2", []uuid.UUID{id2}, 1},
		{"id1, id2", []uuid.UUID{id1, id2}, 3},
		{"nil", []uuid.UUID{}, 0},
	}

	for _, v := range cases {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			assert := assert.New(t)
			devs, err := repo.GetDeviceTokens(set.UUIDSetFromArray(v.users))
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
