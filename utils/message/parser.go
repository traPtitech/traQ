package message

import (
	"regexp"
	"strings"

	"github.com/gofrs/uuid"
	jsonIter "github.com/json-iterator/go"
)

const embURLRegexFragment = `/(files|messages)/[\da-f]{8}-[\da-f]{4}-[\da-f]{4}-[\da-f]{4}-[\da-f]{12}`

var (
	embJSONRegex = regexp.MustCompile(`(?m)!({(?:[ \t\n]*"(?:[^"]|\\.)*"[ \t\n]*:[ \t\n]*"(?:[^"]|\\.)*",)*(?:[ \t\n]*"(?:[^"]|\\.)*"[ \t\n]*:[ \t\n]*"(?:[^"]|\\.)*")})`)
	embURLRegex  = regexp.MustCompile("http://localhost:3000" + embURLRegexFragment)
)

// setMetadataOrigin configures legacy attachment and citation URLs.
func setMetadataOrigin(origin string) {
	embURLRegex = regexp.MustCompile(regexp.QuoteMeta(origin) + embURLRegexFragment)
}

// ParseResult メッセージパースリザルト
type ParseResult struct {
	notificationText string
	PlainText        string
	Mentions         []uuid.UUID
	GroupMentions    []uuid.UUID
	ChannelLink      []uuid.UUID
	Attachments      []uuid.UUID
	Citation         []uuid.UUID
}

// NotificationText returns the Rust-rendered notification as one line.
func (pr *ParseResult) NotificationText() string {
	return strings.ReplaceAll(pr.notificationText, "\n", " ")
}

// parseMetadata preserves the existing bot/search plain text and attachment/citation
// contract. Markdown notifications and references are produced by Rust in Parse.
func parseMetadata(m string) *ParseResult {
	var r ParseResult

	// json型埋め込み
	tmp := embJSONRegex.ReplaceAllStringFunc(m, func(s string) string {
		var info struct {
			Raw  string    `json:"raw"`
			Type string    `json:"type"`
			ID   uuid.UUID `json:"id"`
		}

		if err := jsonIter.ConfigFastest.Unmarshal([]byte(s[1:]), &info); err != nil {
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
	tmp = embURLRegex.ReplaceAllStringFunc(tmp, func(s string) string {
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
