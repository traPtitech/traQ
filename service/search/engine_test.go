package search

import (
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestQueryValidateSort(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		query   string
		invalid bool
	}{
		{"", false},
		{"sort=", false},
		{"sort=null", false},
		{"sort=createdAt", false},
		{"sort=%2BcreatedAt", false},
		{"sort=-updatedAt", false},
		{"sort=xxcreatedAtyy", true},
		{"sort=createdAt%2CupdatedAt", true},
		{"sort=--createdAt", true},
		{"sort=createdAt%0A", true},
	} {
		t.Run(tc.query, func(t *testing.T) {
			t.Parallel()

			c := echo.New().NewContext(httptest.NewRequest("GET", "/?"+tc.query, nil), httptest.NewRecorder())
			var query Query
			require.NoError(t, c.Bind(&query))

			if tc.invalid {
				require.Error(t, query.Validate())
			} else {
				require.NoError(t, query.Validate())
				if !query.Sort.Valid {
					require.Equal(t, "createdAt:desc", query.GetSortKey())
				}
			}
		})
	}
}
