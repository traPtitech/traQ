package message

import (
	"strings"

	jsonIter "github.com/json-iterator/go"
)

// EmbeddedInfo メッセージの埋め込み情報
type EmbeddedInfo struct {
	Raw  string `json:"raw"`
	Type string `json:"type"`
	ID   string `json:"id"`
}

// ExtractEmbedding メッセージの埋め込み情報を抽出したものと、平文化したメッセージを返します
func ExtractEmbedding(m string) (res []*EmbeddedInfo, plain string) {
	res = make([]*EmbeddedInfo, 0)
	tmp := embJSONRegex.ReplaceAllStringFunc(m, func(s string) string {
		info := &EmbeddedInfo{}
		if err := jsonIter.ConfigFastest.Unmarshal([]byte(s[1:]), info); err != nil || len(info.Type) == 0 || len(info.ID) == 0 {
			return s
		}
		res = append(res, info)
		if info.Type == "file" {
			return "[添付ファイル]"
		}
		if info.Type == "message" {
			return "[引用メッセージ]"
		}
		return info.Raw
	})
	return res, strings.ReplaceAll(tmp, "\n", " ")
}
