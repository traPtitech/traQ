package ogpparser

import (
	"fmt"
	"io"
	"net/url"
	"runtime"
	"strings"
	"testing"

	"github.com/dyatlov/go-opengraph/opengraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testHTML = `
<html>
	<head>
		<meta property="og:type" content="article" />
		<meta property="og:title" content="TITLE" />
		<meta property="og:url" content="https://example.com" />
		<meta property="og:image" content="/image.png" />
	</head>
	<body></body>
</html>
`

const testHTMLWithoutOgp = `
<html>
	<head>
		<meta property="og:type" content="article" />
		<meta content="DESCRIPTION" name="description">
		<meta href="https://example.com" name="canonical">
		<meta content="/image.png" itemprop="image">
		<title>TITLE</title>
	</head>
	<body></body>
</html>
`

const testHTMLOgpTagInBody = `
<html>
	<head>
		<title>TITLE</title>
	</head>
	<body>
		<meta property="og:type" content="article" />
	</body>
</html>
`

const testHTMLWithEscapedContents = `
<html>
	<head>
		<meta property="og:type" content="website" />
		<meta property="og:description" content="4種類のコースにて&amp;quot;現場で働くクリエイター&amp;quot;による講義を開催します。">
	</head>
	<body></body>
</html>
`

func parseHTMLString(t *testing.T, h string) (*opengraph.OpenGraph, *DefaultPageMeta) {
	t.Helper()
	og, meta, err := parseBody(strings.NewReader(h), "text/html; charset=utf-8")
	require.NoError(t, err)
	return og, meta
}

func TestParseBody(t *testing.T) {
	t.Parallel()
	t.Run("correct OGP", func(t *testing.T) {
		t.Parallel()
		og, _ := parseHTMLString(t, testHTML)

		assert.Equal(t, "TITLE", og.Title)
		assert.Equal(t, "https://example.com", og.URL)
		assert.Equal(t, "/image.png", og.Images[0].URL)
	})
	t.Run("incorrect OGP", func(t *testing.T) {
		t.Parallel()
		og, meta := parseHTMLString(t, testHTMLWithoutOgp)

		assert.Equal(t, "", og.Title)
		assert.Equal(t, "", og.URL)
		assert.Equal(t, "TITLE", meta.Title)
		assert.Equal(t, "DESCRIPTION", meta.Description)
		assert.Equal(t, "/image.png", meta.Image)
		assert.Equal(t, "https://example.com", meta.URL)
	})
	t.Run("OGP tag in body", func(t *testing.T) {
		t.Parallel()
		og, meta := parseHTMLString(t, testHTMLOgpTagInBody)

		assert.Equal(t, "article", og.Type)
		assert.Equal(t, "TITLE", meta.Title)
	})
	t.Run("HTML with escaped contents", func(t *testing.T) {
		t.Parallel()
		og, _ := parseHTMLString(t, testHTMLWithEscapedContents)

		assert.Equal(t, "website", og.Type)
		assert.Equal(t, "4種類のコースにて\"現場で働くクリエイター\"による講義を開催します。", og.Description)
	})
	t.Run("title", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			html string
			want string
		}{
			{"simple", "<title>TITLE</title>", "TITLE"},
			{"escaped", "<title>A &amp; B</title>", "A & B"},
			{"empty", "<title></title>", ""},
			{"first one", "<title>FIRST</title><title>SECOND</title>", "FIRST"},
			{"skip empty", "<title></title><title>SECOND</title>", "SECOND"},
			{"no title", `<meta content="DESCRIPTION" name="description">`, ""},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, meta := parseHTMLString(t, tt.html)
				assert.Equal(t, tt.want, meta.Title)
			})
		}
	})
	t.Run("ignore meta in noscript", func(t *testing.T) {
		t.Parallel()
		og, _ := parseHTMLString(t, `<noscript><meta property="og:title" content="NOSCRIPT"></noscript>`)

		assert.Equal(t, "", og.Title)
	})
	t.Run("ignore meta in script", func(t *testing.T) {
		t.Parallel()
		og, _ := parseHTMLString(t, `<script>document.write('<meta property="og:title" content="SCRIPT">')</script>`)

		assert.Equal(t, "", og.Title)
	})
	t.Run("huge body", func(t *testing.T) {
		t.Parallel()
		const head = `<html><head><meta property="og:title" content="TITLE" /><title>META TITLE</title></head><body>`
		body := &infiniteReader{pattern: []byte("<p>")}
		og, meta, err := parseBody(io.MultiReader(strings.NewReader(head), body), "text/html; charset=utf-8")

		assert.NoError(t, err)
		assert.Equal(t, "TITLE", og.Title)
		assert.Equal(t, "META TITLE", meta.Title)
		assert.LessOrEqual(t, len(head)+body.read, maxBodySize)
	})
}

// infiniteReader 同じバイト列を無限に返し、読まれたバイト数を記録するReader
type infiniteReader struct {
	pattern []byte
	read    int
}

func (r *infiniteReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.pattern[(r.read+i)%len(r.pattern)]
	}
	r.read += len(p)
	return len(p), nil
}

// TestParseBodyMemory DOMツリーを構築するとメモリ使用量が爆発する入力でも、メモリ使用量が抑えられていることを確認する
//
// 正確に計測するため、他のテストと並列に実行しない
func TestParseBodyMemory(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("<html><head><title>TITLE</title></head><body><p>")
	// 属性が異なる書式要素は Noah's Ark clause で除去されず、テキストが来る度に全て複製される
	for i := range 500 {
		fmt.Fprintf(&sb, "<b id=%d>", i)
	}
	// html.Parse の場合、この時点 (約40KB) で約800MB消費する
	sb.WriteString(strings.Repeat("<p>x", 10000))
	h := sb.String()

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, meta, err := parseBody(strings.NewReader(h), "text/html; charset=utf-8")
	runtime.ReadMemStats(&after)

	require.NoError(t, err)
	assert.Equal(t, "TITLE", meta.Title)
	assert.Less(t, after.TotalAlloc-before.TotalAlloc, uint64(16<<20))
}

func TestFetchTwitterOGP(t *testing.T) {
	// Xのページが高確率で、5～10秒ほどのレイテンシの後503を返してくる仕様になっており、
	// テストがまともにできなくなってしまっているためスキップする
	t.SkipNow()

	t.Parallel()
	tests := []struct {
		name    string
		url     string
		want    func(t *testing.T, res *opengraph.OpenGraph)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success",
			url:  "https://twitter.com/traPtitech/status/1690533645923287040",
			want: func(t *testing.T, res *opengraph.OpenGraph) {
				assert.Equal(t, "設営完了しました！\n西え-33aにてお待ちしています！\n#C102", res.Description)
			},
			wantErr: assert.NoError,
		},
		{
			name: "not found",
			url:  "https://twitter.com/traPtitech/status/1690533645923287041",
			want: func(t *testing.T, res *opengraph.OpenGraph) {
				assert.Equal(t, "", res.Description)
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			u, _ := url.Parse(tt.url)
			got, _, err := ParseMetaForURL(u)
			if !tt.wantErr(t, err, fmt.Sprintf("ParseMetaForURL(%v)", tt.url)) {
				return
			}
			tt.want(t, got)
		})
	}
}
