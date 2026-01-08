package parser

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/dyatlov/go-opengraph/opengraph/types/image"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/imaging"
	"github.com/traPtitech/traQ/utils/optional"
)

// MergeDefaultPageMetaAndOpenGraph OGPの結果とページのメタデータを合わせ、レスポンスの型に揃えます
func MergeDefaultPageMetaAndOpenGraph(og *opengraph.OpenGraph, meta *DefaultPageMeta) *model.Ogp {
	result := &model.Ogp{
		Type:        "website",
		Title:       meta.Title,
		URL:         meta.URL,
		Images:      nil,
		Description: meta.Description,
		Videos:      nil,
	}

	if len(og.Type) > 0 {
		result.Type = og.Type
	}
	if len(og.Title) > 0 {
		result.Title = og.Title
	} else if len(og.SiteName) > 0 {
		result.Title = og.SiteName
	}
	if len(og.URL) > 0 {
		if strings.HasPrefix(og.URL, "/") {
			if metaURL, err := url.Parse(meta.URL); err == nil {
				// 絶対パスではあったがホストなどが含まれていないときに付与する
				result.URL = metaURL.Scheme + "://" + metaURL.Host + og.URL
			} else {
				result.URL = og.URL
			}
		} else {
			result.URL = og.URL
		}
	}
	result.Images = make([]model.OgpMedia, len(og.Images))
	for i, image := range og.Images {
		result.Images[i] = toOgpMedia(image)
	}
	if len(og.Description) > 0 {
		result.Description = og.Description
	}
	result.Videos = make([]model.OgpMedia, len(og.Videos))
	for i, video := range og.Videos {
		// Videoは仕様上ImageのFieldを包含している
		result.Videos[i] = toOgpMedia(&image.Image{
			URL:       video.URL,
			SecureURL: video.SecureURL,
			Type:      video.Type,
			Width:     video.Width,
			Height:    video.Height,
		})
	}

	return result
}

func toOgpMedia(image *image.Image) model.OgpMedia {
	result := model.OgpMedia{
		URL: image.URL,
	}

	if len(image.SecureURL) > 0 {
		result.SecureURL = optional.From(image.SecureURL)
	}
	if len(image.Type) > 0 {
		result.Type = optional.From(image.Type)
	}

	url := image.SecureURL
	if len(url) == 0 {
		url = image.URL
	}

	// Width / Height が未設定の場合は実画像からサイズを取得して fallback する
	width := image.Width
	height := image.Height
	if width == 0 && height == 0 && len(url) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if w, h, err := imaging.FetchImageSize(ctx, &client, url); err == nil {
			width = uint64(w)
			height = uint64(h)
		}
	}

	if width > 0 {
		result.Width = optional.From(int(width))
	}
	if height > 0 {
		result.Height = optional.From(int(height))
	}

	return result
}
