package ogp

import (
	"github.com/dyatlov/go-opengraph/opengraph"
	"net/url"
)

func FetchSpecialDomainInfo(url *url.URL) (og *opengraph.OpenGraph, meta *DefaultPageMeta, isSpecialDomain bool, err error) {
	if url.Host == "twitter.com" {
		og, meta, err = FetchTwitterInfo(url)
		return og, meta, true, err
	}
	return nil, nil, false, nil
}
