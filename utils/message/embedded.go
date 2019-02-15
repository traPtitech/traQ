package message

import (
	"encoding/json"
	"regexp"
	"strings"
)

var embRegex = regexp.MustCompile(`(?m)!({(?:[ \t\n]*"(?:[^"]|\\.)*"[ \t\n]*:[ \t\n]*"(?:[^"]|\\.)*",)*(?:[ \t\n]*"(?:[^"]|\\.)*"[ \t\n]*:[ \t\n]*"(?:[^"]|\\.)*")})`)

// EmbeddedInfo メッセージの埋め込み情報
type EmbeddedInfo struct {
	Raw  string `json:"raw"`
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Parse メッセージの埋め込み情報を抽出したものと、平文化したメッセージを返します
func Parse(m string) (res []*EmbeddedInfo, plain string) {
	tmp := embRegex.ReplaceAllStringFunc(m, func(s string) string {
		info := &EmbeddedInfo{}
		if err := json.Unmarshal([]byte(s[1:]), info); err != nil || len(info.Type) == 0 || len(info.ID) == 0 {
			return s
		}
		res = append(res, info)
		if info.Type == "file" {
			return "file"
		}
		return info.Raw
	})
	return res, strings.Replace(tmp, "\n", " ", -1)
}
