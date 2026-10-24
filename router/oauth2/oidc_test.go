package oauth2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
)

func TestOIDCDiscoveryOmitsUnsupportedLogout(t *testing.T) {
	t.Parallel()
	h := &Handler{Config: Config{Origin: "https://example.com"}}
	res := httptest.NewRecorder()
	require.NoError(t, h.OIDCDiscovery(echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), res)))
	require.Equal(t, http.StatusOK, res.Code)
	var config map[string]any
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &config))
	require.NotContains(t, config, "end_session_endpoint")
	require.Equal(t, "https://example.com/api/v3/users/me/oidc", config["userinfo_endpoint"])
}
