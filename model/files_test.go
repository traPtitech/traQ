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

// TestParallelGroup10 並列テストグループ10 競合がないようなサブテストにすること
func TestParallelGroup10(t *testing.T) {
	assert, _, user, _ := beforeTest(t)

	// File.Create
	t.Run("TestFile_Create", func(t *testing.T) {
		t.Parallel()

		writeData := bytes.NewReader(([]byte)("test message"))

		assert.Error((&File{}).Create(writeData))
		assert.Error((&File{ID: uuid.NewV4()}).Create(writeData))

		file := &File{
			Name:      "testFile.txt",
			Size:      writeData.Size(),
			CreatorID: user.ID,
		}
		assert.NoError(file.Create(writeData))
	})

	// DeleteFile
	t.Run("TestDeleteFile", func(t *testing.T) {
		t.Parallel()

		file := mustMakeFile(t, user.ID)

		assert.NoError(DeleteFile(file.ID))
		_, err := fs.OpenFileByKey(file.getKey())
		assert.Error(err)
	})

	// OpenFileByKey
	t.Run("TestOpenFileByID", func(t *testing.T) {
		t.Parallel()

		f := mustMakeFile(t, user.ID)
		file, err := OpenFileByID(f.ID)
		if assert.NoError(err) {
			defer file.Close()

			buf := make([]byte, 512)
			n, err := file.Read(buf)
			if assert.NoError(err) {
				assert.Equal("test message", string(buf[:n]))
			}
		}
	})

	// GetMetaFileDataByID
	t.Run("TestGetMetaFileDataByID", func(t *testing.T) {
		t.Parallel()

		file := mustMakeFile(t, user.ID)
		result, err := GetMetaFileDataByID(file.ID)
		if assert.NoError(err) {
			assert.Equal(file.ID, result.ID)
		}

		_, err = GetMetaFileDataByID(uuid.Nil)
		assert.Error(err)
	})
}
