package imaging

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenerateIcon(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)

	icon1 := GenerateIcon("a")
	icon2 := GenerateIcon("b")
	icon3 := GenerateIcon("b")

	if assert.NotNil(icon1) && assert.NotNil(icon2) && assert.NotNil(icon3) {
		assert.NotEqual(icon1, icon2)
		assert.EqualValues(icon2, icon3)
	}
}
