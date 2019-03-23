package model

import (
	"fmt"
	"github.com/gofrs/uuid"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFile_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "files", (&File{}).TableName())
}

func TestFile_GetKey(t *testing.T) {
	t.Parallel()
	id := uuid.Must(uuid.NewV4())
	assert.EqualValues(t, id.String(), (&File{ID: id}).GetKey())
}

func TestFile_GetThumbKey(t *testing.T) {
	t.Parallel()
	id := uuid.Must(uuid.NewV4())
	assert.EqualValues(t, fmt.Sprintf("%s-thumb", id.String()), (&File{ID: id}).GetThumbKey())
}

func TestFileACLEntry_TableName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "files_acl", (&FileACLEntry{}).TableName())
}
