package message

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gofrs/uuid"
)

var (
	// ユーザーとグループのnameの和集合
	mentionRegex    = regexp.MustCompile(`[@＠](\S{1,32})`)
	userStartsRegex = regexp.MustCompile(`^[@＠]([a-zA-Z0-9_-]{1,32})`)
	channelRegex    = regexp.MustCompile(`[#＃]([a-zA-Z0-9_/-]+)`)
)

const (
	backQuoteRune          = rune('`')
	dollarRune             = rune('$')
	defaultCodeTokenLength = 3
)

// ReplaceMapper メッセージ埋め込み置換マッピング
type ReplaceMapper interface {
	// Channel チャンネルパス(lower-case) -> チャンネルUUID
	Channel(path string) (uuid.UUID, bool)
	// Group グループ名 -> グループUUID
	Group(name string) (uuid.UUID, bool)
	// User ユーザーID(lower-case) -> ユーザーUUID
	User(name string) (uuid.UUID, bool)
}

// Replacer メッセージ埋め込み置換機
type Replacer struct {
	mapper ReplaceMapper
}

// NewReplacer Replacerを生成します
func NewReplacer(mapper ReplaceMapper) *Replacer {
	return &Replacer{mapper: mapper}
}

// Replace 埋め込みを置換します
func (re *Replacer) Replace(m string) string {
	inCodeBlock := false
	inLatexBlock := false
	codeTokenLength := defaultCodeTokenLength

	lines := strings.Split(m, "\n")
	for i, line := range lines {
		if !inLatexBlock && strings.HasPrefix(line, strings.Repeat("`", codeTokenLength)) {
			// `の数が一致するものと組み合うようにする
			if !inCodeBlock {
				codeTokenLength = countPrefix(line, backQuoteRune)
			} else {
				codeTokenLength = defaultCodeTokenLength
			}

			inCodeBlock = !inCodeBlock
		}
		if !inCodeBlock && strings.HasPrefix(line, "$$") {
			inLatexBlock = !inLatexBlock
		}
		if inCodeBlock || inLatexBlock {
			continue
		}
		// 「```」のブロックでも「$$」ブロック内でもないときに置換

		chs := []rune(line)
		newChs := make([]rune, 0, len(chs))
		// 「`」「$」で囲まれていないところの始めの文字のindex
		noExpressionStartIndex := 0
		for i := 0; i < len(chs); i++ {
			ch := chs[i]
			if ch != backQuoteRune && ch != dollarRune {
				continue
			}

			// 囲まれていない場所が終了したのでその箇所は置換する
			newChs = append(newChs, []rune(
				re.replaceAll(
					string(chs[noExpressionStartIndex:i]),
				),
			)...)

			if ch == dollarRune {
				// 「`」は「$」よりも優先されるので
				// 「$ ` $」のように「`」がペアの「$」より前にあるときは
				// 「$」のペアとして処理しない
				backQuoteI := indexOf(chs[i+1:], backQuoteRune)
				dollarI := indexOf(chs[i+1:], dollarRune)
				if backQuoteI != -1 && dollarI != -1 && backQuoteI < dollarI {
					newChs = append(newChs, ch)
					noExpressionStartIndex = i + 1
					continue
				}
			}
			newI := indexOf(chs[i+1:], ch)
			if newI == -1 {
				// 「$」/「`」のペアがないとき
				newChs = append(newChs, ch)
				noExpressionStartIndex = i + 1
				continue
			}
			newI += i + 1
			newChs = append(newChs, chs[i:newI]...)
			i = newI
			noExpressionStartIndex = newI
		}
		// 最後のペア以降の置換
		newChs = append(newChs, []rune(
			re.replaceAll(
				string(chs[noExpressionStartIndex:]),
			),
		)...)
		lines[i] = string(newChs)
	}
	return strings.Join(lines, "\n")
}

func (re *Replacer) replaceAll(m string) string {
	return re.replaceMention(re.replaceChannel(m))
}

func (re *Replacer) replaceMention(m string) string {
	return mentionRegex.ReplaceAllStringFunc(m, func(s string) string {
		name := strings.ToLower(strings.TrimLeft(s, "@＠"))

		if uid, ok := re.mapper.User(name); ok {
			return fmt.Sprintf(`!{"type":"user","raw":"%s","id":"%s"}`, s, uid)
		}
		if gid, ok := re.mapper.Group(name); ok {
			return fmt.Sprintf(`!{"type":"group","raw":"%s","id":"%s"}`, s, gid)
		}

		if strings.Contains(name, ":") {
			return s
		}
		return userStartsRegex.ReplaceAllStringFunc(s, func(s string) string {
			name := strings.ToLower(strings.TrimLeft(s, "@＠"))

			if uid, ok := re.mapper.User(name); ok {
				return fmt.Sprintf(`!{"type":"user","raw":"%s","id":"%s"}`, s, uid)
			}
			return s
		})
	})
}

func (re *Replacer) replaceChannel(m string) string {
	return channelRegex.ReplaceAllStringFunc(m, func(s string) string {
		c := strings.ToLower(strings.TrimLeft(s, "#＃"))
		if cid, ok := re.mapper.Channel(c); ok {
			return fmt.Sprintf(`!{"type":"channel","raw":"%s","id":"%s"}`, s, cid)
		}
		return s
	})
}

func indexOf(slice []rune, target rune) int {
	for k, v := range slice {
		if v == target {
			return k
		}
	}
	return -1
}

func countPrefix(line string, letter rune) int {
	count := 0
	for _, ch := range line {
		if ch != letter {
			break
		}
		count++
	}
	return count
}
