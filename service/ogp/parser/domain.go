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

const userAgent = "traq-ogp-fetcher; contact: github.com/traPtitech/traQ"

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
