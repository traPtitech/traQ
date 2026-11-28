package search

import "strings"

// Simple query string query の機能のうちどれを有効にするのかを指定する定数。
//
// 本来 " はフレーズ検索のための記法。
// 除外した NEAR(別名SLOP) は、フレーズの直後に「~N」演算子を使用できるようにする機能。N は、トークンの並びにどれだけ位置のずれを許容するかを指定する。
// 引用符で囲んだ文字列を入力どおりに探すため、NEAR と SLOP は含めない。
//
// 同じ ~N 記法のあいまい検索は有効のままにするので、FUZZY は含めてある。 あいまい検索は、トークン内の違いを許容するので、例えば 作家~1 で検索すると、途中に文字が一つ入った「作曲家」にもマッチする。
//
// https://www.elastic.co/guide/en/elasticsearch/reference/8.17/query-dsl-simple-query-string-query.html#supported-flags-values
const simpleQueryFlagsWithoutNear = "AND|NOT|OR|PREFIX|PHRASE|PRECEDENCE|ESCAPE|WHITESPACE|FUZZY"

func rewriteLiteralQuoteQuery(query string) string {
	var rewritten strings.Builder
	rewritten.Grow(len(query))
	for i := 0; i < len(query); i++ {
		c := query[i]
		if c == '\\' && i+1 < len(query) {
			rewritten.WriteString(query[i : i+2])
			i++
			continue
		}
		if c != '"' {
			rewritten.WriteByte(c)
			continue
		}

		end := findClosingQuote(query, i+1)
		if end == -1 {
			rewritten.WriteByte(' ')
			continue
		}
		if end == i+1 {
			rewritten.WriteByte(' ')
			i = end
			continue
		}

		rewritten.WriteByte('"')
		for j := i + 1; j < end; j++ {
			c = query[j]
			if c == '\\' && j+1 < end && isEscapableInQuote(query[j+1]) {
				j++
				c = query[j]
			}
			// Lucene also scans parentheses while finding the end of a group.
			switch c {
			case '\\', '"', '(', ')':
				rewritten.WriteByte('\\')
			}
			rewritten.WriteByte(c)
		}
		rewritten.WriteByte('"')
		i = end
	}
	return rewritten.String()
}

func findClosingQuote(query string, start int) int {
	for i := start; i < len(query); i++ {
		if query[i] == '"' {
			return i
		}
		if query[i] == '\\' && i+1 < len(query) && isEscapableInQuote(query[i+1]) {
			i++
		}
	}
	return -1
}

func isEscapableInQuote(c byte) bool {
	return c == '\\' || c == '"'
}
