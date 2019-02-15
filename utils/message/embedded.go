package message

import (
	"bytes"
	"encoding/json"
	"strings"
)

// EmbeddedInfo メッセージの埋め込み情報
type EmbeddedInfo struct {
	Raw  string `json:"raw"`
	Type string `json:"type"`
	ID   string `json:"id"`
}

// Parse メッセージの埋め込み情報を抽出したものと、平文化したメッセージを返します
// FIXME ""に囲われていない'}'を探すようにする
func Parse(m string) (res []*EmbeddedInfo, plain string) {
	b := strings.Builder{}
	b.Grow(len(m))

	state := 0
	enclosed := bytes.Buffer{}
	for _, r := range m {
		switch state {
		case '!':
			switch r {
			case '{':
				state = '{'
				enclosed.WriteRune(r)
			default:
				state = 0
				b.WriteRune('!')
				b.WriteRune(r)
			}
		case '{':
			switch r {
			case '}':
				enclosed.WriteRune(r)
				info := &EmbeddedInfo{}
				arr := enclosed.Bytes()
				if err := json.Unmarshal(arr, info); err != nil {
					b.WriteRune('!')
					b.Write(arr)
				} else {
					if len(info.Type) == 0 || len(info.ID) == 0 {
						b.WriteRune('!')
						b.Write(arr)
					} else {
						if info.Type == "file" {
							b.WriteString("file")
						} else {
							b.WriteString(info.Raw)
						}
						res = append(res, info)
					}
				}
				enclosed.Reset()
				state = 0
			default:
				enclosed.WriteRune(r)
			}
		default:
			switch r {
			case '!':
				state = '!'
			default:
				b.WriteRune(r)
			}
		}
	}

	if enclosed.Len() > 0 {
		b.WriteRune('!')
		b.Write(enclosed.Bytes())
	}

	return res, strings.Replace(b.String(), "\n", " ", -1)
}
