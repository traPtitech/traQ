package message

import "strings"

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
	cont := true
	for cont {
		state := 0
		contents := []spoilerToken{}
		start := -1
	L:
		for i := range tokens {
			switch state {
			case 0:
				if tokens[i].tType != "C" {
					start = i
					state = 1
				}
				break
			case 1:
				if tokens[i].tType == "C" {
					state = 2
					contents = append(contents, tokens[i])
				} else {
					start = i
					state = 1
				}
				break
			case 2:
				if tokens[i].tType == "C" {
					contents = append(contents, tokens[i])
				} else if tokens[i].tType == "S" {
					contents = []spoilerToken{}
					start = i
					state = 1
				} else {
					// start から start + len(contents) + 1 までを入れ替える
					clength := 0
					for _, t := range tokens {
						clength += len(t.body)
					}

					new := make([]spoilerToken, len(tokens))
					copy(new, tokens)
					new = append(new[:start], spoilerToken{tType: "C", body: strings.Repeat("*", clength)})
					new = append(new, tokens[start+len(contents)+1:]...)
					tokens = new

					contents = []spoilerToken{}
					state = 0
					start = -1
					cont = true
					break L
				}
			}
			cont = false
		}

	}

	return tokens
}

// FillSpoiler メッセージのSpoilerをパースし、塗りつぶします
func FillSpoiler(m string) string {
	return "todo"
}
