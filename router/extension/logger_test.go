package extension

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
)

func TestGetRequestID(t *testing.T) {
	t.Parallel()

	for _, header := range []string{"", "client-request-id"} {
		t.Run(header, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(echo.HeaderXRequestID, header)
			c := echo.New().NewContext(req, httptest.NewRecorder())

			id := GetRequestID(c)
			assert.NotEmpty(t, id)
			if header != "" {
				assert.Equal(t, header, id)
			}

			assert.Equal(t, id, GetRequestID(c))
		})
	}
}
