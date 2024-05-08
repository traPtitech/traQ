package imaging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateIcon(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	icon1, err := GenerateIcon("a")
	require.NoError(t, err)
	icon2, err := GenerateIcon("b")
	require.NoError(t, err)
	icon3, err := GenerateIcon("b")
	require.NoError(t, err)

	if assert.NotNil(icon1) && assert.NotNil(icon2) && assert.NotNil(icon3) {
		assert.NotEqual(icon1, icon2)
		assert.EqualValues(icon2, icon3)
	}
}
