package parser

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/dyatlov/go-opengraph/opengraph/types/image"
	jsonIter "github.com/json-iterator/go"
)

const (
	vrChatAPIBasePath = "https://vrchat.com/api/1"
	vrChatAPIKey      = "JlE5Jldo5Jibnk5O5hTx6XVqsJu4WJ26"
)

type VRChatAPIWorldResponse struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	ImageURL          string `json:"imageUrl"`
	ThumbnailImageURL string `json:"thumbnailImageUrl"`
}

func FetchVRChatInfo(url *url.URL) (*opengraph.OpenGraph, *DefaultPageMeta, error) {
	splitPath := strings.Split(url.Path, "/")

	if len(splitPath) >= 4 && splitPath[1] == "home" && splitPath[2] == "world" && strings.HasPrefix(splitPath[3], "wrld_") {
		worldID := splitPath[3]
		info, err := fetchVRChatWorldInfo(worldID)
		if err != nil {
			return nil, nil, err
		}

		og := opengraph.OpenGraph{
			Title:       fmt.Sprintf("%s - VRChat", info.Name),
			Description: info.Description,
			URL:         url.String(),
			Images: []*image.Image{{
				URL: info.ThumbnailImageURL,
			}},
		}
		meta := DefaultPageMeta{}
		return &og, &meta, nil
	}

	return nil, nil, ErrDomainRequest
}

func fetchVRChatWorldInfo(worldID string) (*VRChatAPIWorldResponse, error) {
	requestURL := fmt.Sprintf("%s/worlds/%s?apiKey=%s", vrChatAPIBasePath, worldID, vrChatAPIKey)
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, ErrNetwork
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return nil, ErrServer
	} else if resp.StatusCode >= 400 {
		return nil, ErrClient
	}

	var data VRChatAPIWorldResponse
	if err = jsonIter.ConfigFastest.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}
