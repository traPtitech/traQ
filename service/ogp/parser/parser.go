package ogpparser

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/dyatlov/go-opengraph/opengraph"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/net/html/charset"
	"golang.org/x/sync/semaphore"
)

const concurrentRequestLimit = 10

// maxBodySize パース対象とするレスポンスボディの最大バイト数
//
// 巨大なページを全て読み込むとメモリや時間を無駄に消費するため、先頭のみを読み込んでパースする。
// OGP のメタタグは通常ページの先頭付近にある。
const maxBodySize = 2 << 20 // 2MiB

var requestLimiter = semaphore.NewWeighted(concurrentRequestLimit)

type DefaultPageMeta struct {
	Title, Description, URL, Image string
}

// ParseMetaForURL 指定したURLのメタタグをパースした結果を返します。
func ParseMetaForURL(url *url.URL) (*opengraph.OpenGraph, *DefaultPageMeta, error) {
	_ = requestLimiter.Acquire(context.Background(), 1)
	defer requestLimiter.Release(1)

	og, meta, isSpecialDomain, err := FetchSpecialDomainInfo(url)
	if isSpecialDomain && (err == nil) {
		return og, meta, nil
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, nil, ErrNetwork
	}

	req.Header.Add("user-agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, ErrNetwork
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, nil, ErrServer
	} else if resp.StatusCode >= 400 {
		return nil, nil, ErrClient
	}

	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/html") {
		return nil, nil, ErrContentTypeNotSupported
	}

	og, meta, err = parseBody(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, nil, err
	}
	if len(meta.URL) == 0 {
		meta.URL = url.String()
	}
	return og, meta, nil
}

// parseBody レスポンスボディの先頭 maxBodySize バイトをHTMLとしてパース
func parseBody(body io.Reader, contentType string) (*opengraph.OpenGraph, *DefaultPageMeta, error) {
	// Decode charset to UTF-8
	decodedReader, err := charset.NewReader(io.LimitReader(body, maxBodySize), contentType)
	if err != nil {
		return nil, nil, ErrParse
	}

	og := opengraph.NewOpenGraph()
	meta := DefaultPageMeta{}
	z := html.NewTokenizer(decodedReader)
	inTitle := false
	for {
		switch z.Next() {
		case html.ErrorToken:
			if errors.Is(z.Err(), io.EOF) {
				return og, &meta, nil
			}
			return nil, nil, ErrParse

		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			switch atom.Lookup(name) {
			case atom.Meta:
				m := make(map[string]string)
				for hasAttr {
					var key, val []byte
					key, val, hasAttr = z.TagAttr()
					m[string(key)] = html.UnescapeString(string(val))
				}
				og.ProcessMeta(m)
				meta.processMeta(m)
			case atom.Title:
				// 最初の title を採用する
				inTitle = len(meta.Title) == 0
			}

		case html.EndTagToken:
			inTitle = false

		case html.TextToken:
			if inTitle {
				meta.Title = string(z.Text())
				inTitle = false
			}
		}
	}
}

// processMeta メタタグ内の情報をパースする
func (m *DefaultPageMeta) processMeta(metaAttrs map[string]string) {
	switch metaAttrs["name"] {
	case "description":
		m.Description = metaAttrs["content"]
	case "canonical":
		m.URL = metaAttrs["href"]
	}
	switch metaAttrs["itemprop"] {
	case "image":
		m.Image = metaAttrs["content"]
	}
}
