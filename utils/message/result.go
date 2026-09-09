package message

import (
	"github.com/gofrs/uuid"
	markdown "github.com/traq-markdown-parser/traq/go"
)

// EmbeddedInfo is an embedding extracted by the Rust preset.
type EmbeddedInfo = markdown.EmbeddedInfo

// ParseResult adapts the Rust processing result to traQ event and bot payloads.
type ParseResult struct {
	Embeddings       []*EmbeddedInfo
	PlainText        string
	Mentions         []uuid.UUID
	GroupMentions    []uuid.UUID
	ChannelLink      []uuid.UUID
	Attachments      []uuid.UUID
	Citation         []uuid.UUID
	notificationText string
}

// NotificationText returns the notification text formatted by the Rust preset.
func (pr *ParseResult) NotificationText() string { return pr.notificationText }
