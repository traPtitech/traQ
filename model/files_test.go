package model

import (
	"database/sql/driver"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFile_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "files", (&FileMeta{}).TableName())
}

func TestFileThumbnail_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "files_thumbnails", (&FileThumbnail{}).TableName())
}

func TestFileACLEntry_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "files_acl", (&FileACLEntry{}).TableName())
}

func TestFileType_Value(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		_, err := FileType(-1).Value()
		assert.Error(t, err)
	})
	t.Run("success", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			fileType FileType
			expected driver.Value
		}{
			{FileTypeUserFile, ""},
			{FileTypeIcon, "icon"},
			{FileTypeStamp, "stamp"},
			{FileTypeThumbnail, "thumbnail"},
			{FileTypeSoundboardItem, "soundboard_item"},
		}

		for _, c := range cases {
			v, err := c.fileType.Value()
			assert.NoError(t, err)
			assert.Equal(t, c.expected, v)
		}
	})
}

func TestFileType_Scan(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src      string
		expected FileType
	}{
		{"", FileTypeUserFile},
		{"icon", FileTypeIcon},
		{"stamp", FileTypeStamp},
		{"thumbnail", FileTypeThumbnail},
		{"soundboard_item", FileTypeSoundboardItem},
	}

	t.Run("error (string)", func(t *testing.T) {
		t.Parallel()

		var f FileType
		assert.Error(t, f.Scan("non_existent"))
	})
	t.Run("error ([]byte)", func(t *testing.T) {
		t.Parallel()

		var f FileType
		assert.Error(t, f.Scan([]byte("non_existent")))
	})
	t.Run("string", func(t *testing.T) {
		t.Parallel()

		for _, c := range cases {
			var f FileType
			assert.NoError(t, f.Scan(c.src))
			assert.Equal(t, c.expected, f)
		}
	})
	t.Run("[]byte", func(t *testing.T) {
		t.Parallel()

		for _, c := range cases {
			var f FileType
			assert.NoError(t, f.Scan([]byte(c.src)))
			assert.Equal(t, c.expected, f)
		}
	})
}

func TestThumbnailType_Value(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		_, err := ThumbnailType(-1).Value()
		assert.Error(t, err)
	})
	t.Run("success", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			fileType ThumbnailType
			expected driver.Value
		}{
			{ThumbnailTypeImage, "image"},
			{ThumbnailTypeWaveform, "waveform"},
		}

		for _, c := range cases {
			v, err := c.fileType.Value()
			assert.NoError(t, err)
			assert.Equal(t, c.expected, v)
		}
	})
}

func TestThumbnailType_Scan(t *testing.T) {
	t.Parallel()

	cases := []struct {
		src      string
		expected ThumbnailType
	}{
		{"image", ThumbnailTypeImage},
		{"waveform", ThumbnailTypeWaveform},
	}

	t.Run("error (string)", func(t *testing.T) {
		t.Parallel()

		var f ThumbnailType
		assert.Error(t, f.Scan("non_existent"))
	})
	t.Run("error ([]byte)", func(t *testing.T) {
		t.Parallel()

		var f ThumbnailType
		assert.Error(t, f.Scan([]byte("non_existent")))
	})
	t.Run("string", func(t *testing.T) {
		t.Parallel()

		for _, c := range cases {
			var f ThumbnailType
			assert.NoError(t, f.Scan(c.src))
			assert.Equal(t, c.expected, f)
		}
	})
	t.Run("[]byte", func(t *testing.T) {
		t.Parallel()

		for _, c := range cases {
			var f ThumbnailType
			assert.NoError(t, f.Scan([]byte(c.src)))
			assert.Equal(t, c.expected, f)
		}
	})
}

func TestThumbnailType_Suffix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		thumbType ThumbnailType
		expected  string
	}{
		{ThumbnailTypeImage, "thumb"},
		{ThumbnailTypeWaveform, "waveform"},
		{ThumbnailType(-1), "null"},
	}
	for _, c := range cases {
		assert.Equal(t, c.expected, c.thumbType.Suffix())
	}
}
