package v3

import (
	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/ogp"
	"github.com/traPtitech/traQ/utils/optional"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
)

type GetOgpRequest struct {
	Url string `json:"url"`
}

// GetOgp GET /ogp?url={url}
func (h* Handlers) GetOgp(c echo.Context) error {
	u, err := url.Parse(c.QueryParam(consts.ParamUrl))
	if err != nil || len(u.Scheme) == 0 || len(u.Host) == 0 {
		return herror.BadRequest("invalid url")
	}

	og, meta, err := parseOpenGraphForUrl(u)
	if err != nil {
		return herror.BadRequest(og)
	}

	merged := mergeDefaultPageMetaToOpenGraph(og, meta)
	return c.JSON(http.StatusOK, merged)
}

func parseOpenGraphForUrl(url *url.URL) (*opengraph.OpenGraph, *ogp.DefaultPageMeta, error) {
	og := opengraph.NewOpenGraph()
	doc, err := fetchMetaForUrl(url)
	if err != nil {
		return nil, nil, err
	}
	og, meta := ogp.ParseDoc(doc)

	return og, meta, nil
}

func fetchMetaForUrl(url *url.URL) (*html.Node, error) {
	resp, err := http.Get(url.String())
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return html.Parse(resp.Body)
}

// mergeDefaultPageMetaToOpenGraph OGPの結果とページのメタデータを合わせ、レスポンスの型に揃えます
func mergeDefaultPageMetaToOpenGraph(og *opengraph.OpenGraph, meta *ogp.DefaultPageMeta) Ogp {
	result := Ogp{
		Type:        "website",
		Title:       meta.Title,
		URL:         meta.Url,
		Images:      nil,
		Description: meta.Description,
		Videos:      nil,
	}

	if len(og.Type) > 0 {
		result.Type = og.Type
	}
	if len(og.Title) > 0 {
		result.Title = og.Title
	}
	if len(og.URL) > 0 {
		result.URL = og.URL
	}
	if len(og.Images) > 0 {
		result.Images = make([]OgpMedia, len(og.Images))
		for i, image := range og.Images {
			result.Images[i] = toOgpMedia(image)
		}
	}
	if len(og.Description) > 0 {
		result.Description = og.Description
	}
	if len(og.Videos) > 0 {
		result.Videos = make([]OgpMedia, len(og.Images))
		for i, video := range og.Videos {
			// Videoは仕様上Imageと同じ構造を持つ
			result.Videos[i] = toOgpMedia((*opengraph.Image)(video))
		}
	}

	return result
}

func toOgpMedia(image *opengraph.Image) OgpMedia {
	result := OgpMedia{
		URL:       image.URL,
		SecureURL: optional.NewString("", false),
		Type:      optional.NewString("", false),
	}

	if len(image.SecureURL) > 0 {
		result.SecureURL = optional.StringFrom(image.SecureURL)
	}
	if len(image.Type) > 0 {
		result.Type = optional.StringFrom(image.Type)
	}
	if image.Width > 0 {
		result.Width = optional.IntFrom(int64(image.Width))
	}
	if image.Height > 0 {
		result.Height = optional.IntFrom(int64(image.Height))
	}

	return result
}
