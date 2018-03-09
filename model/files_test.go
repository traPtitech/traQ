package model

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFile_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "files", (&File{}).TableName())
}

func TestFile_Create(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	writeData := bytes.NewReader(([]byte)("test message"))

	assert.Error((&File{}).Create(writeData))
	assert.Error((&File{ID: CreateUUID()}).Create(writeData))

	file := &File{
		Name:      "testFile.txt",
		Size:      writeData.Size(),
		CreatorID: user.ID,
	}
	if assert.NoError(file.Create(writeData)) {
		fm := NewDevFileManager()

		assert.NotEmpty(file.ID)
		_, err := os.Stat(fm.GetDir() + "/" + file.ID)
		assert.NoError(err)
	}
}

func TestFile_Exists(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	f := mustMakeFile(t, user.ID)
	r := &File{ID: f.ID}

	ok, err := r.Exists()
	if assert.NoError(err) {
		assert.True(ok)
	}

	r2 := &File{ID: CreateUUID()}

	ok, err = r2.Exists()
	if assert.NoError(err) {
		assert.False(ok)
	}
}

func TestFile_Delete(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	file := mustMakeFile(t, user.ID)
	file.IsDeleted = true

	assert.NoError(file.Delete())
}

func TestGetFileByID(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	f := mustMakeFile(t, user.ID)
	file, err := OpenFileByID(f.ID)
	assert.NoError(err)
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)

	if assert.NoError(err) {
		assert.Equal("test message", string(buf[:n]))
	}
}

func TestGetMetaFileDataByID(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	file := mustMakeFile(t, user.ID)
	result, err := GetMetaFileDataByID(file.ID)
	if assert.NoError(err) {
		assert.Equal(file.ID, result.ID)
	}

	none, err := GetMetaFileDataByID("wrongID")
	assert.NoError(err)
	assert.Nil(none)
}
