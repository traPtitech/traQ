package ogp

import (
	"github.com/dyatlov/go-opengraph/opengraph"
	"golang.org/x/net/html"
)

type DefaultPageMeta struct {
	Title, Description, Url, Image string
}

// processMeta メタタグ内の情報をパースする
func (m *DefaultPageMeta) processMeta(metaAttrs map[string]string) {
	switch metaAttrs["name"] {
	case "description":
		m.Description = metaAttrs["content"]
	case "canonical":
		m.Url = metaAttrs["href"]
	}
	switch metaAttrs["itemprop"] {
	case "image":
		m.Image = metaAttrs["content"]
	}
}

// ParseDoc html全体をパース
func ParseDoc(doc *html.Node) (*opengraph.OpenGraph, *DefaultPageMeta) {
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
			parseMetaTags(og, meta, c)  // YouTubeなどへの対応
			break
		}
		parseNode(og, meta, c)
	}
}

// parseMetaTags metaタグを直下の子に持つタグをパース
func parseMetaTags(og *opengraph.OpenGraph, meta *DefaultPageMeta, node *html.Node)  {
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

