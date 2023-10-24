package parser

import (
	"net/http"
	"net/url"
	"time"

	"github.com/dyatlov/go-opengraph/opengraph"
)

var client = http.Client{
	Timeout: 5 * time.Second,
}

// X(Twitter)のOGPを取得するのにuserAgentの中にbotという文字列が入っている必要がある
// Spotifyの新しいOGPを取得するのにuserAgentの中にcurl-botという文字列が入っている必要がある
const userAgent = "traq-ogp-fetcher-curl-bot; contact: github.com/traPtitech/traQ"

func FetchSpecialDomainInfo(url *url.URL) (og *opengraph.OpenGraph, meta *DefaultPageMeta, isSpecialDomain bool, err error) {
	switch url.Host {
	// case "twitter.com":
	// 	og, meta, err = FetchTwitterInfo(url)
	// 	return og, meta, true, err
	case "vrchat.com":
		og, meta, err = FetchVRChatInfo(url)
		return og, meta, true, err
	}
	return nil, nil, false, nil
}
