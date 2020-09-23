package ogp

import (
	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/traPtitech/traQ/model"
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
		result.URL = og.URL
	}
	if len(og.Images) > 0 {
		result.Images = make([]model.OgpMedia, len(og.Images))
		for i, image := range og.Images {
			result.Images[i] = toOgpMedia(image)
		}
	}
	if len(og.Description) > 0 {
		result.Description = og.Description
	}
	if len(og.Videos) > 0 {
		result.Videos = make([]model.OgpMedia, len(og.Videos))
		for i, video := range og.Videos {
			// Videoは仕様上Imageと同じ構造を持つ
			result.Videos[i] = toOgpMedia((*opengraph.Image)(video))
		}
	}

	return result
}

func toOgpMedia(image *opengraph.Image) model.OgpMedia {
	result := model.OgpMedia{
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
