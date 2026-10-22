package search

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/utils/optional"
)

func TestQuerySort(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		sort optional.Of[string]
		want string
	}{
		{optional.Of[string]{}, "createdAt:desc"},
		{optional.From("createdAt"), "createdAt:desc"},
		{optional.From("+createdAt"), "createdAt:desc"},
		{optional.From("-createdAt"), "createdAt:asc"},
		{optional.From("updatedAt"), "updatedAt:desc"},
		{optional.From("+updatedAt"), "updatedAt:desc"},
		{optional.From("-updatedAt"), "updatedAt:asc"},
	} {
		t.Run(tc.sort.ValueOrZero(), func(t *testing.T) {
			t.Parallel()
			q := Query{Sort: tc.sort}
			require.NoError(t, q.Validate())
			require.Equal(t, tc.want, q.GetSortKey())
		})
	}
	for _, sort := range []string{"", "xxcreatedAtyy", "createdAt,updatedAt", "--createdAt", "createdAt\n"} {
		t.Run("invalid/"+sort, func(t *testing.T) {
			t.Parallel()
			require.Error(t, Query{Sort: optional.From(sort)}.Validate())
		})
	}
}
