package v3

import (
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/ogp"
	"net/http"
	"net/url"
)

type GetOgpRequest struct {
	Url string `json:"url"`
}

// GetOgp GET /ogp?url={url}
func (h *Handlers) GetOgp(c echo.Context) error {
	u, err := url.Parse(c.QueryParam(consts.ParamUrl))
	if err != nil || len(u.Scheme) == 0 || len(u.Host) == 0 {
		return herror.BadRequest("invalid url")
	}

	og, meta, err := ogp.ParseMetaForUrl(u)
	if err != nil {
		return herror.BadRequest(og)
	}

	merged := ogp.MergeDefaultPageMetaAndOpenGraph(og, meta)
	return c.JSON(http.StatusOK, merged)
}
