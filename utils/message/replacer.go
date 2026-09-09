package message

import (
	"context"

	"github.com/gofrs/uuid"
	markdown "github.com/traq-markdown-parser/traq/go"
)

// ReplaceMapper resolves application identities for the Rust embedding plan.
type ReplaceMapper interface {
	Channel(path string) (uuid.UUID, bool)
	Group(name string) (uuid.UUID, bool)
	User(name string) (uuid.UUID, bool)
}

type Replacer struct {
	mapper ReplaceMapper
}

func NewReplacer(mapper ReplaceMapper) *Replacer {
	return &Replacer{mapper: mapper}
}

// Replace delegates syntax and replacement rules to the shared Markdown SDK.
func (re *Replacer) Replace(ctx context.Context, source string) (string, error) {
	result, err := processMarkdown(ctx, source)
	if err != nil {
		return "", err
	}

	return markdown.EmbedReferences(source, result.Embedding, re.resolve)
}

func (re *Replacer) resolve(kind markdown.LookupKind, name string) (string, bool) {
	var id uuid.UUID
	var found bool

	switch kind {
	case "user":
		id, found = re.mapper.User(name)
	case "group":
		id, found = re.mapper.Group(name)
	case "channel":
		id, found = re.mapper.Channel(name)
	}

	return id.String(), found
}
