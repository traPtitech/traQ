package oauth2

import (
	"github.com/stretchr/testify/assert"
	"github.com/traPtitech/traQ/oauth2/scope"
	"testing"
)

func TestSplitAndValidateScope(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	{
		s, err := SplitAndValidateScope("read write")
		if assert.NoError(err) {
			assert.True(s.Contains(scope.Read))
			assert.True(s.Contains(scope.Write))
			assert.False(s.Contains(scope.OpenID))
		}
	}
	{
		_, err := SplitAndValidateScope("read write (っ‘△‘ｃ)＜ﾜｧ!")
		assert.Error(err)
	}
	{
		_, err := SplitAndValidateScope("read write read write")
		assert.Error(err)
	}
}
