package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStamp_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "stamps", (&Stamp{}).TableName())
}

func TestFavoriteStamp_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "favorite_stamps", (&FavoriteStamp{}).TableName())
}
