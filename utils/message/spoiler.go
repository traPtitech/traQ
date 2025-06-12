package message

import (
	"strings"
)

type spoilerToken struct {
	tType spoilerTokenType
	body  []rune
}

type spoilerTokenType int

const (
	spoilerTokenInvalid spoilerTokenType = iota
	spoilerTokenExclamation
	spoilerTokenContent
	spoilerTokenSplit
)

func tokenizeSpoiler(msg string) []spoilerToken {
	msgRunes := []rune(msg)
	msgLen := len(msgRunes)
	result := []spoilerToken{}
	contentStartIndex := 0

	for i := 0; i < msgLen; i++ {
		r := msgRunes[i]
		switch r {
		case '!':
			// 連続する!をトークンに変換
			c := countPrefixRune(msgRunes[i:], '!')
			if c >= 2 {
				if i != contentStartIndex {
					result = append(result, spoilerToken{tType: spoilerTokenContent, body: msgRunes[contentStartIndex:i]})
				}

				for _ = range c / 2 {
					result = append(result, spoilerToken{tType: spoilerTokenExclamation, body: msgRunes[i : i+2]})
				}
				i += c - 1
				if c%2 == 1 { // !が奇数個だった場合最後のものは処理していない
					i--
				}
				contentStartIndex = i + 1
			}
		case '\r', '\n', ' ', '　':
			if i != contentStartIndex {
				result = append(result, spoilerToken{tType: spoilerTokenContent, body: msgRunes[contentStartIndex:i]})
			}
			result = append(result, spoilerToken{tType: spoilerTokenSplit, body: msgRunes[i : i+1]})
			contentStartIndex = i + 1
		}
	}

	if msgLen != contentStartIndex {
		result = append(result, spoilerToken{tType: spoilerTokenContent, body: msgRunes[contentStartIndex:msgLen]})
	}
	return result
}

func tokensToString(tokens []spoilerToken) string {
	spoilerStartPos := []int{}
	spoilerEndPos := []int{}

	tokensLen := len(tokens)
	for i, current := range tokens {
		var prev spoilerToken
		var next spoilerToken
		if i > 0 {
			prev = tokens[i-1]
		}
		if i+1 < tokensLen {
			next = tokens[i+1]
		}

		if current.tType == spoilerTokenExclamation {
			if len(spoilerStartPos) > len(spoilerEndPos) {
				if prev.tType != spoilerTokenInvalid &&
					prev.tType != spoilerTokenSplit &&
					spoilerStartPos[len(spoilerStartPos)-1] != i-1 {
					// 閉じれたら閉じる
					spoilerEndPos = append(spoilerEndPos, i)
				} else if next.tType != spoilerTokenInvalid && next.tType != spoilerTokenSplit {
					// 閉じれなくても開けたら開く
					spoilerStartPos = append(spoilerStartPos, i)
				}
			} else {
				if next.tType != spoilerTokenInvalid && next.tType != spoilerTokenSplit {
					spoilerStartPos = append(spoilerStartPos, i)
				}
			}
		}
	}

	// 個数があっていないときは対応関係を正す
	if len(spoilerStartPos) > len(spoilerEndPos) {
		newSpoilerStartPos := make([]int, 0, len(spoilerStartPos))
		for i, j := len(spoilerStartPos)-1, len(spoilerEndPos)-1; i >= 0 && j >= 0; i-- {
			start := spoilerStartPos[i]
			end := spoilerEndPos[j]
			if end < start {
				continue
			}
			newSpoilerStartPos = append(newSpoilerStartPos, start)
			j--
		}

		// newSpoilerStartPosの順番を逆転
		for i := range len(newSpoilerStartPos) / 2 {
			newSpoilerStartPos[i], newSpoilerStartPos[len(newSpoilerStartPos)-i-1] = newSpoilerStartPos[len(newSpoilerStartPos)-i-1], newSpoilerStartPos[i]
		}
		spoilerStartPos = newSpoilerStartPos
	}

	for i := range len(spoilerStartPos) {
		s := spoilerStartPos[i]
		e := spoilerEndPos[i]
		tokens[s].body = nil
		tokens[e].body = nil
		for j := s + 1; j < e; j++ {
			if tokens[j].tType == spoilerTokenSplit && (tokens[j].body[0] == '\r' || tokens[j].body[0] == '\n') {
				continue
			}
			tokens[j].body = []rune(strings.Repeat("█", len(tokens[j].body)))
		}
	}

	result := []rune{}
	for _, v := range tokens {
		result = append(result, v.body...)
	}
	return string(result)
}

func countPrefixRune(line []rune, letter rune) int {
	count := 0
	for _, ch := range line {
		if ch != letter {
			break
		}
		count++
	}
	return count
}

// FillSpoiler メッセージのSpoilerをパースし、塗りつぶします
func FillSpoiler(m string) string {
	return tokensToString(tokenizeSpoiler(m))
}
