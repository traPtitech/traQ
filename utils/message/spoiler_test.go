package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFillSpoiler(t *testing.T) {
	t.Parallel()

	type Case struct {
		Message string
		Filled  string
	}

	cases := []Case{
		{Message: "", Filled: ""},
		{Message: "!", Filled: "!"},
		{Message: "!!", Filled: "!!"},
		{Message: "!!!", Filled: "!!!"},
		{Message: "!!!!", Filled: "!!!!"},
		{Message: "!!!!!", Filled: "!!!!!"},
		{Message: "!! !!", Filled: "!! !!"},
		{Message: "!!Mark!!", Filled: "****"},
		{Message: "x !!!!foo!! bar!!", Filled: "x *******"},
		{Message: "x !!foo !!bar!!!!", Filled: "x *******"},
		{Message: "x !!!!foo!!!!", Filled: "x ***"},
		{Message: "x !!!foo!!!", Filled: "x ****!"},
		{Message: "!!foo !!bar!! baz!!", Filled: "***********"},
		{Message: "foo !! bar !! baz", Filled: "foo !! bar !! baz"},
		{Message: "x !!a !!foo!!!!!!!!!!!bar!! b!!", Filled: "x *****!!******"},
		{Message: "x !!a !!foo!!!!!!!!!!!!bar!! b!!", Filled: "x *****!!!!*****"},
	}

	for _, v := range cases {
		v := v
		t.Run(v.Filled, func(t *testing.T) {
			t.Parallel()
			filled := FillSpoiler(v.Message)
			assert.EqualValues(t, v.Filled, filled)
		})
	}
}
