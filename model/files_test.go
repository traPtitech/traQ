package model

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFile_TableName(t *testing.T) {
	assert.Equal(t, "files", (&File{}).TableName())
}

func TestFile_Create(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)
	writeData := bytes.NewReader(([]byte)("test message"))

	assert.Error((&File{}).Create(writeData))
	assert.Error((&File{ID: CreateUUID()}).Create(writeData))

	file := &File{
		Name:      "testFile.txt",
		Size:      writeData.Size(),
		CreatorID: testUserID,
	}
	if assert.NoError(file.Create(writeData)) {
		assert.NotEmpty(file.ID)
		_, err := os.Stat(dirName + "/" + file.ID)
		assert.NoError(err)
	}
}

func TestFile_Delete(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	file := mustMakeFile(t)
	file.IsDeleted = true

	assert.NoError(file.Delete())
}

func TestGetFileByID(t *testing.T) {
	beforeTest(t)
	assert := assert.New(t)

	f := mustMakeFile(t)
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
	beforeTest(t)
	assert := assert.New(t)

	file := mustMakeFile(t)
	result, err := GetMetaFileDataByID(file.ID)
	if assert.NoError(err) {
		assert.Equal(file.ID, result.ID)
	}

	_, err = GetMetaFileDataByID("wrongID")
	assert.Error(err)
}
