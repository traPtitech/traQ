package v3

import (
	"net/http"
	"testing"
)

func TestHandlers_GetVersion(t *testing.T) {
	t.Parallel()
	env := Setup(t, common)

	e := env.R(t)
	obj := e.GET("/api/v3/version").
		Expect().
		Status(http.StatusOK).
		JSON().
		Object()
	obj.Value("version").String().Equal("version")
	obj.Value("revision").String().Equal("revision")
}
