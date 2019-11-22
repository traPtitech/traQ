package v3

import (
	"net/http"
	"testing"
)

func TestHandlers_GetVersion(t *testing.T) {
	t.Parallel()
	_, server := Setup(t, common)

	e := R(t, server)
	obj := e.GET("/api/v3/version").
		Expect().
		Status(http.StatusOK).
		JSON().
		Object()
	obj.Value("version").String().Equal("version")
	obj.Value("revision").String().Equal("revision")
}
