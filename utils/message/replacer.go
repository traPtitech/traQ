package message

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gofrs/uuid"
)

var (
	// ユーザーとグループのnameの和集合
	mentionRegex    = regexp.MustCompile(`:?[@＠]([^\s@＠]{0,31}[^\s@＠:])`)
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

type replaceState struct {
	inCodeBlock     bool
	inLatexBlock    bool
	codeTokenLength int
}

// NewReplacer Replacerを生成します
func NewReplacer(mapper ReplaceMapper) *Replacer {
	return &Replacer{mapper: mapper}
}

// Replace 埋め込みを置換します
func (re *Replacer) Replace(m string) string {
	lines := strings.Split(m, "\n")
	state := replaceState{codeTokenLength: defaultCodeTokenLength}

	for i, line := range lines {
		lines[i] = re.replaceLine(line, &state)
	}
	return strings.Join(lines, "\n")
}

func (re *Replacer) replaceLine(line string, state *replaceState) string {
	if !state.inLatexBlock && strings.HasPrefix(line, strings.Repeat("`", state.codeTokenLength)) {
		// `の数が一致するものと組み合うようにする
		if state.inCodeBlock {
			state.codeTokenLength = defaultCodeTokenLength
		} else {
			state.codeTokenLength = countPrefix(line, backQuoteRune)
		}
		state.inCodeBlock = !state.inCodeBlock
	}

	if !state.inCodeBlock && strings.HasPrefix(line, "$$") {
		state.inLatexBlock = !state.inLatexBlock
	}
	if state.inCodeBlock || state.inLatexBlock {
		return line
	}

	return re.replaceOutsideExpressions(line)
}

func (re *Replacer) replaceOutsideExpressions(line string) string {
	chars := []rune(line)
	replaced := make([]rune, 0, len(chars))
	outsideStart := 0

	for i := 0; i < len(chars); i++ {
		ch := chars[i]
		if ch != backQuoteRune && ch != dollarRune {
			continue
		}

		// 囲まれていない場所が終了したのでその箇所は置換する
		replaced = append(replaced, []rune(re.replaceAll(string(chars[outsideStart:i])))...)

		if ch == dollarRune {
			// 「`」は「$」よりも優先されるので
			// 「$ ` $」のように「`」がペアの「$」より前にあるときは
			// 「$」のペアとして処理しない
			backQuoteIndex := indexOf(chars[i+1:], backQuoteRune)
			dollarIndex := indexOf(chars[i+1:], dollarRune)
			if backQuoteIndex != -1 && dollarIndex != -1 && backQuoteIndex < dollarIndex {
				replaced = append(replaced, ch)
				outsideStart = i + 1
				continue
			}
		}

		pairIndex := indexOf(chars[i+1:], ch)
		if pairIndex == -1 {
			// 「$」/「`」のペアがないとき
			replaced = append(replaced, ch)
			outsideStart = i + 1
			continue
		}
		pairIndex += i + 1
		replaced = append(replaced, chars[i:pairIndex]...)
		i = pairIndex
		outsideStart = pairIndex
	}

	// 最後のペア以降の置換
	replaced = append(replaced, []rune(re.replaceAll(string(chars[outsideStart:])))...)
	return string(replaced)
}

func (re *Replacer) replaceAll(m string) string {
	return re.replaceMention(re.replaceChannel(m))
}

func (re *Replacer) replaceMention(m string) string {
	return mentionRegex.ReplaceAllStringFunc(m, func(s string) string {
		// 始まりが:なものを除外
		if strings.HasPrefix(s, ":") {
			return s
		}

		name := strings.ToLower(strings.TrimLeft(s, "@＠"))

		if uid, ok := re.mapper.User(name); ok {
			return fmt.Sprintf(`!{"type":"user","raw":"%s","id":"%s"}`, s, uid)
		}
		if gid, ok := re.mapper.Group(name); ok {
			return fmt.Sprintf(`!{"type":"group","raw":"%s","id":"%s"}`, s, gid)
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
