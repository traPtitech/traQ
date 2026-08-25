package gorm

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/utils/optional"
	random2 "github.com/traPtitech/traQ/utils/random"
	"github.com/traPtitech/traQ/utils/set"
)

func TestRepositoryImpl_RegisterDevice(t *testing.T) {
	t.Parallel()
	repo, assert, _ := setup(t, common)

	id1 := mustMakeUser(t, repo, rand, false).GetID()
	id2 := mustMakeUser(t, repo, rand, false).GetID()
	id3 := mustMakeUser(t, repo, rand, true).GetID()
	id4 := mustMakeUser(t, repo, rand, true).GetID()
	id5 := mustMakeUser(t, repo, rand, false).GetID()
	id6 := mustMakeUser(t, repo, rand, false).GetID()
	id7 := mustMakeUser(t, repo, rand, true).GetID()
	id8 := mustMakeUser(t, repo, rand, true).GetID()
	id9 := mustMakeUser(t, repo, rand, false).GetID()
	id10 := mustMakeUser(t, repo, rand, false).GetID()
	id11 := mustMakeUser(t, repo, rand, true).GetID()
	id12 := mustMakeUser(t, repo, rand, true).GetID()

	token1 := random2.AlphaNumeric(20)
	token2 := random2.AlphaNumeric(20)
	token3 := random2.AlphaNumeric(20)
	token4 := random2.AlphaNumeric(20)
	token5 := random2.AlphaNumeric(20)
	token6 := random2.AlphaNumeric(20)
	token7 := random2.AlphaNumeric(20)
	token8 := random2.AlphaNumeric(20)

	fid1 := random2.AlphaNumeric(22)
	fid2 := random2.AlphaNumeric(22)
	fid3 := random2.AlphaNumeric(22)
	fid4 := random2.AlphaNumeric(22)
	fid5 := random2.AlphaNumeric(22)
	fid6 := random2.AlphaNumeric(22)
	fid7 := random2.AlphaNumeric(22)
	fid8 := random2.AlphaNumeric(22)

	cases := []struct {
		user  uuid.UUID
		token repository.RegisterDeviceArgs
		error bool
	}{
		{id1, repository.RegisterDeviceArgs{Token: optional.New(token1, true)}, false},
		{id2, repository.RegisterDeviceArgs{Token: optional.New(token2, true)}, false},
		{id2, repository.RegisterDeviceArgs{Token: optional.New(token2, true)}, false},
		{id3, repository.RegisterDeviceArgs{Token: optional.New(token3, true)}, false},
		{id4, repository.RegisterDeviceArgs{Token: optional.New(token4, true)}, false},
		{id1, repository.RegisterDeviceArgs{Token: optional.New(token2, true)}, true},
		{uuid.Nil, repository.RegisterDeviceArgs{Token: optional.New(token2, true)}, true},
		{id1, repository.RegisterDeviceArgs{Token: optional.New("", true)}, true},
		{id1, repository.RegisterDeviceArgs{Token: optional.New("", false)}, true},

		{id5, repository.RegisterDeviceArgs{FID: optional.New(fid1, true)}, false},
		{id6, repository.RegisterDeviceArgs{FID: optional.New(fid2, true)}, false},
		{id6, repository.RegisterDeviceArgs{FID: optional.New(fid2, true)}, false},
		{id7, repository.RegisterDeviceArgs{FID: optional.New(fid3, true)}, false},
		{id8, repository.RegisterDeviceArgs{FID: optional.New(fid4, true)}, false},
		{id5, repository.RegisterDeviceArgs{FID: optional.New(fid2, true)}, true},
		{uuid.Nil, repository.RegisterDeviceArgs{FID: optional.New(fid2, true)}, true},
		{id5, repository.RegisterDeviceArgs{FID: optional.New("", true)}, true},
		{id5, repository.RegisterDeviceArgs{FID: optional.New("", false)}, true},

		{id9, repository.RegisterDeviceArgs{Token: optional.New(token5, true), FID: optional.New(fid5, true)}, false},
		{id10, repository.RegisterDeviceArgs{Token: optional.New(token6, true), FID: optional.New(fid6, true)}, false},
		{id10, repository.RegisterDeviceArgs{Token: optional.New(token6, true), FID: optional.New(fid6, true)}, false},
		{id11, repository.RegisterDeviceArgs{Token: optional.New(token7, true), FID: optional.New(fid7, true)}, false},
		{id12, repository.RegisterDeviceArgs{Token: optional.New(token8, true), FID: optional.New(fid8, true)}, false},
		{id9, repository.RegisterDeviceArgs{Token: optional.New(token6, true), FID: optional.New(fid6, true)}, true},
		{uuid.Nil, repository.RegisterDeviceArgs{Token: optional.New(token6, true), FID: optional.New(fid6, true)}, true},
		{id9, repository.RegisterDeviceArgs{Token: optional.New("", true), FID: optional.New("", true)}, true},
		{id9, repository.RegisterDeviceArgs{Token: optional.New("", false), FID: optional.New("", false)}, true},
	}

	for _, v := range cases {
		err := repo.RegisterDevice(context.TODO(), v.user, v.token)
		if v.error {
			assert.Error(err)
		} else {
			assert.NoError(err)
		}
	}

	assert.EqualValues(12, count(t, getDB(repo).Model(model.Device{}).Where("user_id IN (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", id1, id2, id3, id4, id5, id6, id7, id8, id9, id10, id11, id12)))
}

func TestRepositoryImpl_DeleteDeviceTokens(t *testing.T) {
	t.Parallel()
	repo, assert, require := setup(t, common)

	id1 := mustMakeUser(t, repo, rand, false).GetID()
	id2 := mustMakeUser(t, repo, rand, false).GetID()
	id3 := mustMakeUser(t, repo, rand, false).GetID()
	id4 := mustMakeUser(t, repo, rand, true).GetID()

	token1 := random2.AlphaNumeric(20)
	token2 := random2.AlphaNumeric(20)
	token3 := random2.AlphaNumeric(20)
	token4 := random2.AlphaNumeric(20)
	token5 := random2.AlphaNumeric(20)
	token6 := random2.AlphaNumeric(20)
	token7 := random2.AlphaNumeric(20)

	err := repo.RegisterDevice(context.TODO(), id1, repository.RegisterDeviceArgs{Token: optional.New(token1, true)})
	require.NoError(err)
	err = repo.RegisterDevice(context.TODO(), id2, repository.RegisterDeviceArgs{Token: optional.New(token2, true)})
	require.NoError(err)
	err = repo.RegisterDevice(context.TODO(), id1, repository.RegisterDeviceArgs{Token: optional.New(token3, true)})
	require.NoError(err)
	err = repo.RegisterDevice(context.TODO(), id1, repository.RegisterDeviceArgs{Token: optional.New(token4, true)})
	require.NoError(err)
	err = repo.RegisterDevice(context.TODO(), id3, repository.RegisterDeviceArgs{Token: optional.New(token5, true)})
	require.NoError(err)
	err = repo.RegisterDevice(context.TODO(), id4, repository.RegisterDeviceArgs{Token: optional.New(token6, true)})
	require.NoError(err)
	err = repo.RegisterDevice(context.TODO(), id4, repository.RegisterDeviceArgs{Token: optional.New(token7, true)})
	require.NoError(err)

	cases := []struct {
		tokens []string
		expect int
	}{
		{[]string{token2}, 6},         // v7単体
		{[]string{token6}, 5},         // v4単体
		{[]string{}, 5},               // 空配列
		{[]string{token1, token5}, 3}, // v7 2つ
		{[]string{token4, token7}, 1}, //v4 とv7 1つずつ
		{[]string{token3, token2, token6}, 0},
	}
	for _, v := range cases {
		assert.NoError(repo.DeleteDeviceTokens(context.TODO(), v.tokens))
		assert.EqualValues(v.expect, count(t, getDB(repo).Model(model.Device{}).Where("user_id IN (?, ?, ?, ?)", id1, id2, id3, id4)))
	}
}

func TestRepositoryImpl_GetDeviceTokens(t *testing.T) {
	t.Parallel()
	repo, _, require := setup(t, common)

	id1 := mustMakeUser(t, repo, rand, false).GetID()
	id2 := mustMakeUser(t, repo, rand, false).GetID()
	id3 := mustMakeUser(t, repo, rand, true).GetID()

	token1 := random2.AlphaNumeric(20)
	token2 := random2.AlphaNumeric(20)
	token3 := random2.AlphaNumeric(20)
	token4 := random2.AlphaNumeric(20)

	err := repo.RegisterDevice(context.TODO(), id1, repository.RegisterDeviceArgs{Token: optional.New(token1, true)})
	require.NoError(err)
	err = repo.RegisterDevice(context.TODO(), id2, repository.RegisterDeviceArgs{Token: optional.New(token2, true)})
	require.NoError(err)
	err = repo.RegisterDevice(context.TODO(), id1, repository.RegisterDeviceArgs{Token: optional.New(token3, true)})
	require.NoError(err)
	err = repo.RegisterDevice(context.TODO(), id3, repository.RegisterDeviceArgs{Token: optional.New(token4, true)})
	require.NoError(err)

	cases := []struct {
		name   string
		users  []uuid.UUID
		expect int
	}{
		{"id1", []uuid.UUID{id1}, 2},
		{"id2", []uuid.UUID{id2}, 1},
		{"id1, id2", []uuid.UUID{id1, id2}, 3},
		{"id3", []uuid.UUID{id3}, 1},
		{"id1, id3", []uuid.UUID{id1, id3}, 3},
		{"nil", []uuid.UUID{}, 0},
	}

	for _, v := range cases {
		v := v
		t.Run(v.name, func(t *testing.T) {
			t.Parallel()
			assert := assert.New(t)
			devs, err := repo.GetDeviceTokens(context.TODO(), set.UUIDSetFromArray(v.users))
			if assert.NoError(err) {
				n := 0
				for _, arr := range devs {
					n += len(arr)
				}
				assert.EqualValues(v.expect, n)
			}
		})
	}
}
