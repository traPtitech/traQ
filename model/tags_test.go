package model

import (
	"github.com/satori/go.uuid"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTag_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "tags", (&Tag{}).TableName())
}

func TestCreateTag(t *testing.T) {
	assert, _, _, _ := beforeTest(t)

	{
		tag, err := CreateTag("tagA", false, "")
		if assert.NoError(err) {
			assert.NotEmpty(tag.ID)
			assert.Equal("tagA", tag.Name)
			assert.False(tag.Restricted)
			assert.Empty(tag.Type)
			assert.NotZero(tag.CreatedAt)
			assert.NotZero(tag.UpdatedAt)
			count := 0
			db.Table("tags").Count(&count)
			assert.Equal(1, count)
		}

	}
	{
		tag, err := CreateTag("tagB", true, "aaaa")
		if assert.NoError(err) {
			assert.NotEmpty(tag.ID)
			assert.Equal("tagB", tag.Name)
			assert.True(tag.Restricted)
			assert.Equal("aaaa", tag.Type)
			assert.NotZero(tag.CreatedAt)
			assert.NotZero(tag.UpdatedAt)
			count := 0
			db.Table("tags").Count(&count)
			assert.Equal(2, count)
		}
	}
}

func TestChangeTagType(t *testing.T) {
	assert, require, _, _ := beforeTest(t)

	tag, err := CreateTag("tagA", false, "")
	require.NoError(err)

	{
		err := ChangeTagType(tag.GetID(), "newType", true)
		if assert.NoError(err) {
			t, err := GetTagByID(tag.GetID())
			require.NoError(err)
			assert.Equal("newType", t.Type)
			assert.True(t.Restricted)
		}
	}
}

func TestGetTagByID(t *testing.T) {
	assert, require, _, _ := beforeTest(t)

	tag, err := CreateTag("tagA", false, "")
	require.NoError(err)

	r, err := GetTagByID(tag.GetID())
	if assert.NoError(err) {
		assert.Equal(tag.Name, r.Name)
	}

	_, err = GetTagByID(uuid.NewV4())
	assert.Error(err)
}

func TestGetTagByName(t *testing.T) {
	assert, require, _, _ := beforeTest(t)

	tag, err := CreateTag("tagA", false, "")
	require.NoError(err)

	r, err := GetTagByName("tagA")
	if assert.NoError(err) {
		assert.Equal(tag.ID, r.ID)
	}

	_, err = GetTagByName("nothing")
	assert.Error(err)

	_, err = GetTagByName("")
	assert.Error(err)
}

func TestGetAllTags(t *testing.T) {
	assert, require, _, _ := beforeTest(t)

	_, err := CreateTag("tagA", false, "")
	require.NoError(err)
	_, err = CreateTag("tagB", false, "")
	require.NoError(err)
	_, err = CreateTag("tagC", false, "")
	require.NoError(err)
	_, err = CreateTag("tagD", false, "")
	require.NoError(err)

	tags, err := GetAllTags()
	if assert.NoError(err) {
		assert.Len(tags, 4)
	}
}
