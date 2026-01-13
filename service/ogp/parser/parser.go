package parser

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dyatlov/go-opengraph/opengraph"
	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/sync/semaphore"
)

const concurrentRequestLimit = 10

var requestLimiter = semaphore.NewWeighted(concurrentRequestLimit)

type DefaultPageMeta struct {
	Title, Description, URL, Image string
}

// isPrivateIP はIPアドレスがプライベート、ループバック、リンクローカル、またはその他の内部アドレスかどうかを判定します
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true // 不明なIPはブロック
	}
	// ループバック (127.0.0.0/8, ::1)
	if ip.IsLoopback() {
		return true
	}
	// プライベートアドレス (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, fc00::/7)
	if ip.IsPrivate() {
		return true
	}
	// リンクローカル (169.254.0.0/16, fe80::/10)
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// 未指定アドレス (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return true
	}
	// マルチキャスト
	if ip.IsMulticast() {
		return true
	}
	return false
}

// validateURL はURLがSSRFに対して安全かどうかを検証し、検証済みのIPアドレスを返します
func validateURL(u *url.URL) ([]net.IP, error) {
	// スキームの検証
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, ErrNotAllowed
	}

	// ホスト名を取得
	host := u.Hostname()
	if host == "" {
		return nil, ErrNotAllowed
	}

	// IPアドレスの場合は直接検証
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateIP(ip) {
			return nil, ErrNotAllowed
		}
		return []net.IP{ip}, nil
	}

	// ホスト名の場合はDNS解決して検証
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, ErrNetwork
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return nil, ErrNotAllowed
		}
	}

	return ips, nil
}

// ParseMetaForURL 指定したURLのメタタグをパースした結果を返します。
func ParseMetaForURL(url *url.URL) (*opengraph.OpenGraph, *DefaultPageMeta, error) {
	_ = requestLimiter.Acquire(context.Background(), 1)
	defer requestLimiter.Release(1)

	// SSRF対策: URLを検証し、検証済みIPアドレスを取得
	resolvedIPs, err := validateURL(url)
	if err != nil {
		return nil, nil, err
	}

	og, meta, isSpecialDomain, err := FetchSpecialDomainInfo(url)
	if isSpecialDomain && (err == nil) {
		return og, meta, nil
	}

	// SSRF対策: 検証済みのIPアドレスを使用してリクエストを送信
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			// 検証済みのIPアドレスに接続
			addr = net.JoinHostPort(resolvedIPs[0].String(), port)
			return dialer.DialContext(ctx, network, addr)
		},
	}

	client := http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			// Validate redirect destination to prevent SSRF via redirects
			_, err := validateURL(req.URL)
			return err
		},
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

	// Decode charset to UTF-8
	decodedReader, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
	if err != nil {
		return nil, nil, ErrParse
	}
	doc, err := html.Parse(decodedReader)
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
				m[a.Key] = html.UnescapeString(a.Val)
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
