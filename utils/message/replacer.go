package message

import (
	"fmt"
	"github.com/gofrs/uuid"
	"regexp"
	"strings"
)

var (
	mentionRegex = regexp.MustCompile(`[@＠]([\S]+)`)
	channelRegex = regexp.MustCompile(`[#＃]([a-zA-Z0-9_/-]+)`)
)

type Replacer struct {
	ChannelMap map[string]uuid.UUID
	UserMap    map[string]uuid.UUID
	GroupMap   map[string]uuid.UUID
}

func (re *Replacer) Replace(m string) string {
	inCodeBlock := false
	lines := strings.Split(m, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
		}
		if !inCodeBlock {
			inQuote := false
			split := strings.Split(line, "`")
			for j, s := range split {
				if !inQuote {
					split[j] = re.replaceChannel(re.replaceMention(s))
				}
				inQuote = !inQuote
			}
			lines[i] = strings.Join(split, "`")
		}
	}
	return strings.Join(lines, "\n")
}

func (re *Replacer) replaceMention(m string) string {
	return mentionRegex.ReplaceAllStringFunc(m, func(s string) string {
		t := strings.ToLower(strings.TrimLeft(s, "@＠"))
		if uid, ok := re.UserMap[t]; ok {
			return fmt.Sprintf(`!{"type":"user","raw":"%s","id":"%s"}`, s, uid)
		}
		if gid, ok := re.GroupMap[t]; ok {
			return fmt.Sprintf(`!{"type":"group","raw":"%s","id":"%s"}`, s, gid)
		}
		return s
	})
}

func (re *Replacer) replaceChannel(m string) string {
	return channelRegex.ReplaceAllStringFunc(m, func(s string) string {
		c := strings.ToLower(strings.TrimLeft(s, "#＃"))
		if cid, ok := re.ChannelMap[c]; ok {
			return fmt.Sprintf(`!{"type":"channel","raw":"%s","id":"%s"}`, s, cid)
		}
		return s
	})
}
