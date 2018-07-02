package model

import (
	"bytes"
	"github.com/satori/go.uuid"
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
	assert.NoError(file.Create(writeData))
}

func TestDeleteFile(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	file := mustMakeFile(t, user.ID)

	assert.NoError(DeleteFile(file.GetID()))
	_, err := fileManagers[""].OpenFileByID(file.ID)
	assert.Error(err)
}

func TestOpenFileByID(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	f := mustMakeFile(t, user.ID)
	file, err := OpenFileByID(f.GetID())
	if assert.NoError(err) {
		defer file.Close()

		buf := make([]byte, 512)
		n, err := file.Read(buf)
		if assert.NoError(err) {
			assert.Equal("test message", string(buf[:n]))
		}
	}
}

func TestGetMetaFileDataByID(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	file := mustMakeFile(t, user.ID)
	result, err := GetMetaFileDataByID(file.GetID())
	if assert.NoError(err) {
		assert.Equal(file.ID, result.ID)
	}

	none, err := GetMetaFileDataByID(uuid.Nil)
	if assert.NoError(err) {
		assert.Nil(none)
	}
}
