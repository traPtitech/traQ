package oauth2

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestErrorResponse_Error(t *testing.T) {
	t.Parallel()

	assert.EqualValues(t, errInvalidRequest, &errorResponse{ErrorType: errInvalidRequest})
}
