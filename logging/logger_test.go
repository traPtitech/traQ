package logging

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateNewLogger(t *testing.T) {
	t.Parallel()

	l, err := CreateNewLogger("test", "test")
	assert.NoError(t, err)
	assert.NotNil(t, l)
}
