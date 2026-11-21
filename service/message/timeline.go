package message

import (
	"time"
)

type Timeline interface {
	Query() TimelineQuery
	Records() []DetailedMessage
	HasMore() bool
	RetrievedAt() time.Time
}
