package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDevice_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "devices", (&Device{}).TableName())
}

func TestRegisterDevice(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	id1 := user.GetUID()
	id2 := mustMakeUser(t, "test2").GetUID()
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"

	{
		_, err := RegisterDevice(id1, token1)
		assert.NoError(err)
	}
	{
		_, err := RegisterDevice(id2, token2)
		assert.NoError(err)
	}
	{
		_, err := RegisterDevice(id1, token2)
		assert.Error(err)

	}

	l := 0
	require.NoError(db.Model(Device{}).Count(&l).Error)
	assert.EqualValues(2, l)
}

func TestUnregisterDevice(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	id1 := user.GetUID()
	id2 := mustMakeUser(t, "test2").GetUID()
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token3 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvfawfefwfwe3iqodwjkdvlznfjbxdefpuw90jiosdv"

	{
		_, err := RegisterDevice(id1, token1)
		require.NoError(err)
	}
	{
		_, err := RegisterDevice(id2, token2)
		require.NoError(err)
	}
	{
		_, err := RegisterDevice(id1, token3)
		require.NoError(err)
	}

	{
		assert.NoError(UnregisterDevice(id2, token2))
		l := 0
		require.NoError(db.Model(Device{}).Count(&l).Error)
		assert.EqualValues(2, l)
	}
	{
		assert.NoError(UnregisterDevice(id1, token2))
		l := 0
		require.NoError(db.Model(Device{}).Count(&l).Error)
		assert.EqualValues(2, l)
	}
}

func TestGetAllDevices(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	id1 := user.GetUID()
	id2 := mustMakeUser(t, "test2").GetUID()
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token3 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvfawfefwfwe3iqodwjkdvlznfjbxdefpuw90jiosdv"

	{
		_, err := RegisterDevice(id1, token1)
		require.NoError(err)
	}
	{
		_, err := RegisterDevice(id2, token2)
		require.NoError(err)
	}
	{
		_, err := RegisterDevice(id1, token3)
		require.NoError(err)
	}

	devs, err := GetAllDevices()
	if assert.NoError(err) {
		assert.Len(devs, 3)
	}
}

func TestGetDevices(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	id1 := user.GetUID()
	id2 := mustMakeUser(t, "test2").GetUID()
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token3 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvfawfefwfwe3iqodwjkdvlznfjbxdefpuw90jiosdv"

	{
		_, err := RegisterDevice(id1, token1)
		require.NoError(err)
	}
	{
		_, err := RegisterDevice(id2, token2)
		require.NoError(err)
	}
	{
		_, err := RegisterDevice(id1, token3)
		require.NoError(err)
	}

	devs, err := GetDevices(id1)
	if assert.NoError(err) {
		assert.Len(devs, 2)
	}

	devs, err = GetDevices(id2)
	if assert.NoError(err) {
		assert.Len(devs, 1)
	}
}

func TestGetAllDeviceIds(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	id1 := user.GetUID()
	id2 := mustMakeUser(t, "test2").GetUID()
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token3 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvfawfefwfwe3iqodwjkdvlznfjbxdefpuw90jiosdv"

	{
		_, err := RegisterDevice(id1, token1)
		require.NoError(err)
	}
	{
		_, err := RegisterDevice(id2, token2)
		require.NoError(err)
	}
	{
		_, err := RegisterDevice(id1, token3)
		require.NoError(err)
	}

	devs, err := GetAllDeviceIDs()
	if assert.NoError(err) {
		assert.Len(devs, 3)
	}
}

func TestGetDeviceIds(t *testing.T) {
	assert, require, user, _ := beforeTest(t)

	id1 := user.GetUID()
	id2 := mustMakeUser(t, "test2").GetUID()
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token3 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvfawfefwfwe3iqodwjkdvlznfjbxdefpuw90jiosdv"

	{
		_, err := RegisterDevice(id1, token1)
		require.NoError(err)
	}
	{
		_, err := RegisterDevice(id2, token2)
		require.NoError(err)
	}
	{
		_, err := RegisterDevice(id1, token3)
		require.NoError(err)
	}

	devs, err := GetDeviceIDs(id1)
	if assert.NoError(err) {
		assert.Len(devs, 2)
	}

	devs, err = GetDeviceIDs(id2)
	if assert.NoError(err) {
		assert.Len(devs, 1)
	}
}
