package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTag_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "tags", (&Tag{}).TableName())
}

func TestTag_Create(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	tag := &Tag{
		Name: "Create test",
	}
	if assert.NoError(tag.Create()) {
		assert.NotEmpty(tag.ID)
	}

	assert.Error((&Tag{}).Create())
}

func TestTag_Exists(t *testing.T) {
	assert, require, _, _ := beforeTest(t)

	// 正常系
	tag := &Tag{
		Name: "existTag",
	}
	require.NoError(tag.Create())

	has, err := (&Tag{Name: tag.Name}).Exists()
	if assert.NoError(err) {
		assert.True(has)
	}

	tag = &Tag{
		Name: "wrong tag",
	}

	has, err = tag.Exists()
	if assert.NoError(err) {
		assert.False(has)
	}
}

func TestGetTagByID(t *testing.T) {
	assert, require, _, _ := beforeTest(t)

	// 正常系
	tag := &Tag{
		Name: "getTag",
	}
	require.NoError(tag.Create())

	r, err := GetTagByID(tag.ID)
	if assert.NoError(err) {
		assert.Equal(tag.Name, r.Name)
	}

	_, err = GetTagByID("wrong_id")
	assert.Error(err)
}
