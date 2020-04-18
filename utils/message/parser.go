package message

import (
	"github.com/gofrs/uuid"
	jsoniter "github.com/json-iterator/go"
	"regexp"
	"strings"
)

const embUrlRegexFragment = `/(files|messages)/[\da-f]{8}-[\da-f]{4}-[\da-f]{4}-[\da-f]{4}-[\da-f]{12}`

var (
	embJsonRegex = regexp.MustCompile(`(?m)!({(?:[ \t\n]*"(?:[^"]|\\.)*"[ \t\n]*:[ \t\n]*"(?:[^"]|\\.)*",)*(?:[ \t\n]*"(?:[^"]|\\.)*"[ \t\n]*:[ \t\n]*"(?:[^"]|\\.)*")})`)
	embUrlRegex  = regexp.MustCompile("http://localhost:3000" + embUrlRegexFragment)
)

// SetOrigin URL型埋め込みのURLのオリジンを設定します
func SetOrigin(origin string) {
	embUrlRegex = regexp.MustCompile(strings.ReplaceAll(origin, ".", `\.`) + embUrlRegexFragment)
}

// ParseResult メッセージパースリザルト
type ParseResult struct {
	PlainText     string
	Mentions      []uuid.UUID
	GroupMentions []uuid.UUID
	ChannelLink   []uuid.UUID
	Attachments   []uuid.UUID
	Citation      []uuid.UUID
}

// OneLine PlainTextを１行化したものを返します
func (pr *ParseResult) OneLine() string {
	return strings.Replace(pr.PlainText, "\n", " ", -1)
}

// Parse メッセージをパースし、埋め込み情報を抽出します
func Parse(m string) *ParseResult {
	var r ParseResult

	// json型埋め込み
	tmp := embJsonRegex.ReplaceAllStringFunc(m, func(s string) string {
		var info struct {
			Raw  string    `json:"raw"`
			Type string    `json:"type"`
			ID   uuid.UUID `json:"id"`
		}

		if err := jsoniter.ConfigFastest.Unmarshal([]byte(s[1:]), &info); err != nil {
			return s
		}

		switch info.Type {
		case "file":
			r.Attachments = append(r.Attachments, info.ID)
			return "[添付ファイル]"
		case "message":
			r.Citation = append(r.Citation, info.ID)
			return "[引用メッセージ]"
		case "user":
			r.Mentions = append(r.Mentions, info.ID)
			return info.Raw
		case "group":
			r.GroupMentions = append(r.GroupMentions, info.ID)
			return info.Raw
		case "channel":
			r.ChannelLink = append(r.ChannelLink, info.ID)
			return info.Raw
		default:
			return s
		}
	})

	// url型埋め込み
	tmp = embUrlRegex.ReplaceAllStringFunc(tmp, func(s string) string {
		switch {
		case strings.Contains(s, "/files/"):
			r.Attachments = append(r.Attachments, uuid.FromStringOrNil(s[len(s)-36:]))
			return "[添付ファイル]"
		case strings.Contains(s, "/messages/"):
			r.Citation = append(r.Citation, uuid.FromStringOrNil(s[len(s)-36:]))
			return "[引用メッセージ]"
		default:
			return s
		}
	})

	r.PlainText = tmp
	return &r
}
