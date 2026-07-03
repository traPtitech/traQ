package message

import (
	"time"
)

type Timeline interface {
	Query() TimelineQuery
	Records() []DetailedBecauseAttachmentsAndQuotesAreIncluded
	HasMore() bool
	RetrievedAt() time.Time
}
