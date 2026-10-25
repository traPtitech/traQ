package search

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/utils/optional"
)

func TestQueryValidateSort(t *testing.T) {
	t.Parallel()

	for _, sort := range []optional.Of[string]{
		{}, optional.From("createdAt"), optional.From("+createdAt"), optional.From("-updatedAt"),
	} {
		t.Run(sort.ValueOrZero(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, Query{Sort: sort}.Validate())
		})
	}
	for _, sort := range []string{"", "xxcreatedAtyy", "createdAt,updatedAt", "--createdAt", "createdAt\n"} {
		t.Run("invalid/"+sort, func(t *testing.T) {
			t.Parallel()
			require.Error(t, Query{Sort: optional.From(sort)}.Validate())
		})
	}
}
