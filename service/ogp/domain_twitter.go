package ogp

import (
	"encoding/json"
	"fmt"
	"github.com/dyatlov/go-opengraph/opengraph"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TwitterSyndicationAPIResponse struct {
	Text string `json:"text"`
	User struct {
		Name            string `json:"name"`
		ProfileImageUrl string `json:"profile_image_url"`
		ScreenName      string `json:"screen_name"`
	} `json:"user"`
	Photos []struct {
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"photos"`
	Video struct {
		Poster string `json:"poster"`
	} `json:"video"`
}

func FetchTwitterInfo(url *url.URL) (*opengraph.OpenGraph, *DefaultPageMeta, error) {
	splitPath := strings.Split(url.Path, "/")
	if len(splitPath) < 4 || splitPath[2] != "status" {
		return nil, nil, ErrDomainRequest
	}
	statusId := splitPath[3]
	apiResponse, err := fetchTwitterSyndicationAPI(statusId)
	if err != nil {
		return nil, nil, err
	}
	og := opengraph.OpenGraph{
		Title:       fmt.Sprintf("%s on Twitter", apiResponse.User.Name),
		Description: apiResponse.Text,
		URL:         url.String(),
	}
	result := DefaultPageMeta{}
	if len(apiResponse.Photos) > 0 {
		image := apiResponse.Photos[0]
		og.Images = []*opengraph.Image {{
			URL: image.URL,
			Width: uint64(image.Width),
			Height: uint64(image.Height),
		}}
	} else if apiResponse.Video.Poster != "" {
		og.Images = []*opengraph.Image {{
			URL: apiResponse.Video.Poster,
		}}
	}
	return &og, &result, nil
}

func fetchTwitterSyndicationAPI(statusId string) (*TwitterSyndicationAPIResponse, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	requestUrl := fmt.Sprintf("https://syndication.twitter.com/tweet?id=%s", statusId)
	resp, err := client.Get(requestUrl)
	if err != nil {
		return nil, ErrNetwork
	}

	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return nil, ErrServer
	} else if resp.StatusCode >= 400 {
		return nil, ErrClient
	}

	byteArray, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	data := new(TwitterSyndicationAPIResponse)

	if err = json.Unmarshal(byteArray, data); err != nil {
		return nil, err
	}
	return data, nil
}
