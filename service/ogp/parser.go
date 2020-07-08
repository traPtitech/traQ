package ogp

import (
	"github.com/dyatlov/go-opengraph/opengraph"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type DefaultPageMeta struct {
	Title, Description, URL, Image string
}

// ParseMetaForURL 指定したURLのメタタグをパースした結果を返します。
func ParseMetaForURL(url *url.URL) (*opengraph.OpenGraph, *DefaultPageMeta, error) {
	og, meta, isSpecialDomain, err := FetchSpecialDomainInfo(url)
	if isSpecialDomain && (err == nil) {
		return og, meta, nil
	}

	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(url.String())
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

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, nil, ErrParse
	}

	og, meta = parseDoc(doc)
	if len(meta.URL) == 0 {
		meta.URL = url.String()
	}
	return og, meta, nil
}

// parseDoc html全体をパース
func parseDoc(doc *html.Node) (*opengraph.OpenGraph, *DefaultPageMeta) {
	og := opengraph.NewOpenGraph()
	meta := DefaultPageMeta{}
	parseNode(og, &meta, doc)
	return og, &meta
}

// parseNode ノードを深さ優先でパース
func parseNode(og *opengraph.OpenGraph, meta *DefaultPageMeta, node *html.Node) {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "head" {
			parseMetaTags(og, meta, c)
			continue
		} else if c.Type == html.ElementNode && c.Data == "body" {
			parseMetaTags(og, meta, c) // YouTubeなどへの対応
			break
		}
		parseNode(og, meta, c)
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

// parseMetaTags metaタグを直下の子に持つタグをパース
func parseMetaTags(og *opengraph.OpenGraph, meta *DefaultPageMeta, node *html.Node) {
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && c.Data == "meta" {
			m := make(map[string]string)
			for _, a := range c.Attr {
				m[a.Key] = a.Val
			}
			og.ProcessMeta(m)
			meta.processMeta(m)
		} else if title := extractTitleFromNode(c); len(title) > 0 {
			meta.Title = title
		}
	}
}

func extractTitleFromNode(node *html.Node) string {
	if node.Type == html.ElementNode &&
		node.Data == "title" &&
		node.FirstChild != nil &&
		node.FirstChild.Type == html.TextNode {
		return node.FirstChild.Data
	}
	return ""
}
