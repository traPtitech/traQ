package router

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	v3 "github.com/traPtitech/traQ/router/v3"
)

func TestMeta(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		origin     string
		requestURL string
	}{
		{
			name:       "production",
			origin:     "https://q.example.com",
			requestURL: "https://123-prod.preview.example.com/api/v3/meta",
		},
		{
			name:       "development",
			origin:     "https://q-dev.example.com",
			requestURL: "https://123-dev.preview.example.com/api/v3/meta",
		},
		{
			name:       "local origin with port",
			origin:     "http://localhost:3000",
			requestURL: "http://localhost:5173/api/v3/meta",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			h := &v3.Handlers{Config: provideV3Config(&Config{Origin: tc.origin})}
			h.Setup(e.Group("/api"))

			// Authentication is not required, and proxy headers must not determine the origin.
			req := httptest.NewRequest(http.MethodGet, tc.requestURL, nil)
			req.Header.Set("Forwarded", "host=proxy.example.com;proto=https")
			req.Header.Set("X-Forwarded-Host", "proxy.example.com")
			req.Header.Set("X-Forwarded-Proto", "https")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			require.JSONEq(t, fmt.Sprintf(`{"canonicalOrigin": %q}`, tc.origin), rec.Body.String())
		})
	}
}
