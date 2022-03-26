package parser

import (
	"net/url"

	"github.com/dyatlov/go-opengraph/opengraph"
)

func FetchSpecialDomainInfo(url *url.URL) (og *opengraph.OpenGraph, meta *DefaultPageMeta, isSpecialDomain bool, err error) {
	switch url.Host {
	case "twitter.com":
		og, meta, err = FetchTwitterInfo(url)
		return og, meta, true, err
	case "vrchat.com":
		og, meta, err = FetchVRChatInfo(url)
		return og, meta, true, err
	}
	return nil, nil, false, nil
}
