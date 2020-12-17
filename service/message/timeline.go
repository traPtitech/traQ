package message

import (
	"time"
)

type Timeline interface {
	Query() TimelineQuery
	Records() []Message
	HasMore() bool
	RetrievedAt() time.Time
}
