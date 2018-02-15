package model

import (
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDevice_TableName(t *testing.T) {
	assert.Equal(t, "devices", (&Device{}).TableName())
}

func TestDevice_Register(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	id1 := testUserID
	id2 := privateUserID
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"

	assert.NoError((&Device{UserID: id1, Token: token1}).Register())
	assert.NoError((&Device{UserID: id2, Token: token2}).Register())
	assert.Error((&Device{UserID: id1, Token: token2}).Register())

	l, err := db.Count(&Device{})
	require.NoError(t, err)
	assert.Equal(int64(2), l)
}

func TestDevice_Unregister(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	id1 := testUserID
	id2 := privateUserID
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token3 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvfawfefwfwe3iqodwjkdvlznfjbxdefpuw90jiosdv"

	assert.NoError((&Device{UserID: id1, Token: token1}).Register())
	assert.NoError((&Device{UserID: id2, Token: token2}).Register())
	assert.NoError((&Device{UserID: id1, Token: token3}).Register())

	assert.NoError((&Device{Token: token2}).Unregister())
	l, err := db.Count(&Device{})
	require.NoError(t, err)
	assert.Equal(int64(2), l)

	assert.NoError((&Device{UserID: id1}).Unregister())
	l, err = db.Count(&Device{})
	require.NoError(t, err)
	assert.Equal(int64(0), l)
}

func TestGetAllDevices(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	id1 := testUserID
	id2 := privateUserID
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token3 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvfawfefwfwe3iqodwjkdvlznfjbxdefpuw90jiosdv"

	assert.NoError((&Device{UserID: id1, Token: token1}).Register())
	assert.NoError((&Device{UserID: id2, Token: token2}).Register())
	assert.NoError((&Device{UserID: id1, Token: token3}).Register())

	devs, err := GetAllDevices()
	if assert.NoError(err) {
		assert.Len(devs, 3)
	}
}

func TestGetDevices(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	id1 := testUserID
	id2 := privateUserID
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token3 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvfawfefwfwe3iqodwjkdvlznfjbxdefpuw90jiosdv"

	assert.NoError((&Device{UserID: id1, Token: token1}).Register())
	assert.NoError((&Device{UserID: id2, Token: token2}).Register())
	assert.NoError((&Device{UserID: id1, Token: token3}).Register())

	devs, err := GetDevices(uuid.FromStringOrNil(id1))
	if assert.NoError(err) {
		assert.Len(devs, 2)
	}

	devs, err = GetDevices(uuid.FromStringOrNil(id2))
	if assert.NoError(err) {
		assert.Len(devs, 1)
	}
}

func TestGetAllDeviceIds(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	id1 := testUserID
	id2 := privateUserID
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token3 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvfawfefwfwe3iqodwjkdvlznfjbxdefpuw90jiosdv"

	assert.NoError((&Device{UserID: id1, Token: token1}).Register())
	assert.NoError((&Device{UserID: id2, Token: token2}).Register())
	assert.NoError((&Device{UserID: id1, Token: token3}).Register())

	devs, err := GetAllDeviceIDs()
	if assert.NoError(err) {
		assert.Len(devs, 3)
	}
}

func TestGetDeviceIds(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	id1 := testUserID
	id2 := privateUserID
	token1 := "ajopejiajgopnavdnva8y48fhaerudsyf8uf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token2 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvlfjxhgyru83iqodwjkdvlznfjbxdefpuw90jiosdv"
	token3 := "ajopejiajgopnavdnva8y48ffwefwefewfwf39ifoewkvfawfefwfwe3iqodwjkdvlznfjbxdefpuw90jiosdv"

	assert.NoError((&Device{UserID: id1, Token: token1}).Register())
	assert.NoError((&Device{UserID: id2, Token: token2}).Register())
	assert.NoError((&Device{UserID: id1, Token: token3}).Register())

	devs, err := GetDeviceIDs(uuid.FromStringOrNil(id1))
	if assert.NoError(err) {
		assert.Len(devs, 2)
	}

	devs, err = GetDeviceIDs(uuid.FromStringOrNil(id2))
	if assert.NoError(err) {
		assert.Len(devs, 1)
	}
}
