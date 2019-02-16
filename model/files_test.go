package model

import (
	"fmt"
	"github.com/satori/go.uuid"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFile_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "files", (&File{}).TableName())
}

func TestFile_GetKey(t *testing.T) {
	t.Parallel()
	id := uuid.NewV4()
	assert.EqualValues(t, id.String(), (&File{ID: id}).GetKey())
}

func TestFile_GetThumbKey(t *testing.T) {
	t.Parallel()
	id := uuid.NewV4()
	assert.EqualValues(t, fmt.Sprintf("%s-thumb", id.String()), (&File{ID: id}).GetThumbKey())
}
