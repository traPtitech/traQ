package model

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/utils"
	"testing"
)

func TestDevice_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "devices", (&Device{}).TableName())
}

// TestParallelGroup3 並列テストグループ3 競合がないようなサブテストにすること
func TestParallelGroup3(t *testing.T) {
	assert, require, _, _ := beforeTest(t)

	// RegisterDevice
	t.Run("TestRegisterDevice", func(t *testing.T) {
		t.Parallel()

		id1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).ID
		id2 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).ID
		token1 := utils.RandAlphabetAndNumberString(20)
		token2 := utils.RandAlphabetAndNumberString(20)

		cases := []struct {
			user  uuid.UUID
			token string
			error bool
		}{
			{id1, token1, false},
			{id2, token2, false},
			{id1, token2, true},
		}

		for _, v := range cases {
			_, err := RegisterDevice(v.user, v.token)
			if v.error {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		}

		l := 0
		require.NoError(db.Model(Device{}).Where("user_id IN (?, ?)", id1, id2).Count(&l).Error)
		assert.EqualValues(2, l)
	})

	// UnregisterDevice
	t.Run("TestUnregisterDevice", func(t *testing.T) {
		t.Parallel()

		id1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).ID
		id2 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).ID
		token1 := utils.RandAlphabetAndNumberString(20)
		token2 := utils.RandAlphabetAndNumberString(20)
		token3 := utils.RandAlphabetAndNumberString(20)

		_, err := RegisterDevice(id1, token1)
		require.NoError(err)
		_, err = RegisterDevice(id2, token2)
		require.NoError(err)
		_, err = RegisterDevice(id1, token3)
		require.NoError(err)

		cases := []struct {
			token  string
			expect int
		}{
			{token2, 2},
			{token3, 1},
		}
		for _, v := range cases {
			assert.NoError(UnregisterDevice(v.token))
			l := 0
			require.NoError(db.Model(Device{}).Where("user_id IN (?, ?)", id1, id2).Count(&l).Error)
			assert.EqualValues(v.expect, l)
		}
	})

	// GetDevices
	t.Run("TestGetDevices", func(t *testing.T) {
		t.Parallel()

		id1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).ID
		id2 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).ID
		token1 := utils.RandAlphabetAndNumberString(20)
		token2 := utils.RandAlphabetAndNumberString(20)
		token3 := utils.RandAlphabetAndNumberString(20)

		_, err := RegisterDevice(id1, token1)
		require.NoError(err)
		_, err = RegisterDevice(id2, token2)
		require.NoError(err)
		_, err = RegisterDevice(id1, token3)
		require.NoError(err)

		cases := []struct {
			name   string
			user   uuid.UUID
			expect int
		}{
			{"id1", id1, 2},
			{"id2", id2, 1},
		}

		for _, v := range cases {
			v := v
			t.Run(v.name, func(t *testing.T) {
				t.Parallel()

				devs, err := GetDevices(v.user)
				if assert.NoError(err) {
					assert.Len(devs, v.expect)
				}
			})
		}
	})

	// GetDeviceIds
	t.Run("TestGetDeviceIds", func(t *testing.T) {
		t.Parallel()

		id1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).ID
		id2 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).ID
		token1 := utils.RandAlphabetAndNumberString(20)
		token2 := utils.RandAlphabetAndNumberString(20)
		token3 := utils.RandAlphabetAndNumberString(20)

		_, err := RegisterDevice(id1, token1)
		require.NoError(err)
		_, err = RegisterDevice(id2, token2)
		require.NoError(err)
		_, err = RegisterDevice(id1, token3)
		require.NoError(err)

		cases := []struct {
			name   string
			user   uuid.UUID
			expect int
		}{
			{"id1", id1, 2},
			{"id2", id2, 1},
		}

		for _, v := range cases {
			v := v
			t.Run(v.name, func(t *testing.T) {
				t.Parallel()

				devs, err := GetDeviceIDs(v.user)
				if assert.NoError(err) {
					assert.Len(devs, v.expect)
				}
			})
		}
	})
}

// TestParallelGroup4 並列テストグループ4 競合がないようなサブテストにすること
func TestParallelGroup4(t *testing.T) {
	assert, require, _, _ := beforeTest(t)

	id1 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).ID
	id2 := mustMakeUser(t, utils.RandAlphabetAndNumberString(20)).ID
	token1 := utils.RandAlphabetAndNumberString(20)
	token2 := utils.RandAlphabetAndNumberString(20)
	token3 := utils.RandAlphabetAndNumberString(20)

	_, err := RegisterDevice(id1, token1)
	require.NoError(err)
	_, err = RegisterDevice(id2, token2)
	require.NoError(err)
	_, err = RegisterDevice(id1, token3)
	require.NoError(err)

	// GetAllDevices
	t.Run("TestGetAllDevices", func(t *testing.T) {
		t.Parallel()

		devs, err := GetAllDevices()
		if assert.NoError(err) {
			assert.Len(devs, 3)
		}
	})

	// GetAllDeviceIDs
	t.Run("TestGetAllDeviceIds", func(t *testing.T) {
		t.Parallel()

		devs, err := GetAllDeviceIDs()
		if assert.NoError(err) {
			assert.Len(devs, 3)
		}
	})
}
