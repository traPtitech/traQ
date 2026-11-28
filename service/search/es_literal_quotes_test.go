package search

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/utils/optional"
)

func TestRewriteLiteralQuoteQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		query string
		want  string
	}{
		{"foo~1 | bar*", "foo~1 | bar*"},
		{"\"テキスト", " テキスト"},
		{"前\"後", "前 後"},
		{"\"\"語", " 語"},
		{"\" \"", "\" \""},
		{"\"a  b\"~1", "\"a  b\"~1"},
		{"\"あいうえお\"かきく", "\"あいうえお\"かきく"},
		{"\"東京\" \"渋谷", "\"東京\"  渋谷"},
		{"東京 -(\"中央区\" | \"東銀座\")", "東京 -(\"中央区\" | \"東銀座\")"},
		{"(\"a+b (test)\" | \"a*b\")", "(\"a+b \\(test\\)\" | \"a*b\")"},
		{"\"name=\\\"traQ\\\"\"", "\"name=\\\"traQ\\\"\""},
		{"\"C:\\tmp\\\\\"", "\"C:\\\\tmp\\\\\""},
		{"\\\"word", "\\\"word"},
	}
	for _, test := range tests {
		t.Run(test.query, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.want, rewriteLiteralQuoteQuery(test.query))
		})
	}
}

func TestESEngineDoLiteralQuotes(t *testing.T) {
	t.Parallel()

	channelID := uuid.Must(uuid.NewV4())
	for _, test := range []struct {
		name, word, wantWord string
		wantQuoteSyntax      bool
	}{
		{name: "ordinary search syntax", word: "foo~1 | bar*", wantWord: "foo~1 | bar*"},
		{name: "blank search word", word: " \t", wantWord: " \t"},
		{name: "quoted text", word: "\"a b\"~1", wantWord: "\"a b\"~1", wantQuoteSyntax: true},
		{name: "quoted space", word: "\" \"", wantWord: "\" \"", wantQuoteSyntax: true},
		{name: "unmatched quote with a word", word: "\"foo", wantWord: " foo", wantQuoteSyntax: true},
		{name: "empty quotes", word: "\"\""},
		{name: "negated empty quotes", word: `-""`, wantWord: "- ", wantQuoteSyntax: true},
		{name: "group containing empty quotes", word: `("")`, wantWord: "( )", wantQuoteSyntax: true},
		{name: "unmatched quote", word: "\""},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var requestBody []byte
			client, err := elasticsearch.NewClient(elasticsearch.Config{
				Transport: searchTestTransport(func(req *http.Request) (*http.Response, error) {
					var err error
					requestBody, err = io.ReadAll(req.Body)
					if err != nil {
						return nil, err
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     http.Header{"X-Elastic-Product": []string{"Elasticsearch"}},
						Body:       io.NopCloser(strings.NewReader("{\"hits\":{\"total\":{\"value\":0},\"hits\":[]}}")),
					}, nil
				}),
			})
			require.NoError(t, err)
			engine := &esEngine{client: client, mm: &searchTestMessageManager{}, l: zap.NewNop()}
			_, err = engine.Do(t.Context(), &Query{Word: optional.From(test.word), In: optional.From(channelID)})
			require.NoError(t, err)
			var body struct {
				Query struct {
					Bool struct {
						Must []struct {
							Text *simpleQueryString `json:"simple_query_string"`
							Term termQuery          `json:"term"`
						} `json:"must"`
					} `json:"bool"`
				} `json:"query"`
			}
			require.NoError(t, json.Unmarshal(requestBody, &body))
			must := body.Query.Bool.Must
			require.NotEmpty(t, must)
			assert.Equal(t, termQuery{"channelId": {Value: channelID.String()}}, must[len(must)-1].Term)

			if test.wantWord == "" {
				require.Len(t, must, 1)
				assert.Nil(t, must[0].Text)
				return
			}
			require.Len(t, must, 2)
			got := must[0].Text
			require.NotNil(t, got)
			assert.Equal(t, test.wantWord, got.Query)
			assert.Equal(t, []string{"text"}, got.Fields)
			assert.Equal(t, "AND", got.DefaultOperator)
			if test.wantQuoteSyntax {
				assert.Equal(t, ".literal", got.QuoteFieldSuffix)
				assert.ElementsMatch(t,
					[]string{"AND", "NOT", "OR", "PREFIX", "PHRASE", "PRECEDENCE", "ESCAPE", "WHITESPACE", "FUZZY"},
					strings.Split(got.Flags, "|"),
				)
			} else {
				assert.Empty(t, got.QuoteFieldSuffix)
				assert.Empty(t, got.Flags)
			}
		})
	}
}
