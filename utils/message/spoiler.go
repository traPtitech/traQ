package message

type spoilerToken struct {
	tType string
	body  string
}

func tokenizeSpoiler(msg string) []spoilerToken {
	result := []spoilerToken{}

	excl := false
	space := false
	str := ""
	for _, r := range msg {
		switch r {
		case '!':
			if excl && space {
				result = append(result, spoilerToken{tType: "ExclamationS", body: "!!"})
				excl = false
				space = false
				str = ""
			} else if excl {
				result = append(result, spoilerToken{tType: "Exclamation", body: "!!"})
				excl = false
				str = ""
			} else {
				excl = true
				if str != "" {
					result = append(result, spoilerToken{tType: "Content", body: str})
					str = ""
				}
				str = str + "!"
			}
		case '\r', '\n', ' ', '　':
			space = true
			str = str + string(r)
			if excl {
				excl = false
			}
		default:
			space = false
			str = str + string(r)
			if excl {
				excl = false
			}
		}
	}
	if str != "" {
		result = append(result, spoilerToken{tType: "Content", body: str})
	}

	return result
}

func parseSpoiler(tokens []spoilerToken) []spoilerToken {
	return []spoilerToken{{tType: "a", body: "aa"}}
}

// FillSpoiler メッセージのSpoilerをパースし、塗りつぶします
func FillSpoiler(m string) string {
	return "todo"
}
