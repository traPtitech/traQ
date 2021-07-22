package ogp

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
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

func TestParseDoc(t *testing.T) {
	t.Parallel()
	t.Run("correct OGP", func(t *testing.T) {
		t.Parallel()
		doc, _ := html.Parse(strings.NewReader(testHTML))
		og, _ := parseDoc(doc)

		assert.Equal(t, "TITLE", og.Title)
		assert.Equal(t, "https://example.com", og.URL)
		assert.Equal(t, "/image.png", og.Images[0].URL)
	})
	t.Run("incorrect OGP", func(t *testing.T) {
		t.Parallel()
		doc, _ := html.Parse(strings.NewReader(testHTMLWithoutOgp))
		og, meta := parseDoc(doc)

		assert.Equal(t, "", og.Title)
		assert.Equal(t, "", og.URL)
		assert.Equal(t, "TITLE", meta.Title)
		assert.Equal(t, "DESCRIPTION", meta.Description)
		assert.Equal(t, "/image.png", meta.Image)
		assert.Equal(t, "https://example.com", meta.URL)
	})
	t.Run("OGP tag in body", func(t *testing.T) {
		t.Parallel()
		doc, _ := html.Parse(strings.NewReader(testHTMLOgpTagInBody))
		og, meta := parseDoc(doc)

		assert.Equal(t, "article", og.Type)
		assert.Equal(t, "TITLE", meta.Title)
	})
}

func TestExtractTitleFromNode(t *testing.T) {
	t.Parallel()
	t.Run("Correct title node", func(t *testing.T) {
		t.Parallel()
		const h = "<title>TITLE</title>"
		n, _ := html.Parse(strings.NewReader(h))
		result := extractTitleFromNode(n.FirstChild.FirstChild.FirstChild)

		assert.Equal(t, "TITLE", result)
	})
	t.Run("Incorrect title node (no content)", func(t *testing.T) {
		t.Parallel()
		const h = "<title></title>"
		n, _ := html.Parse(strings.NewReader(h))
		result := extractTitleFromNode(n.FirstChild.FirstChild.FirstChild)

		assert.Equal(t, "", result)
	})
	t.Run("Not a title node", func(t *testing.T) {
		t.Parallel()
		const h = `<meta content="DESCRIPTION" name="description">`
		n, _ := html.Parse(strings.NewReader(h))
		result := extractTitleFromNode(n.FirstChild.FirstChild.FirstChild)

		assert.Equal(t, "", result)
	})
}
