//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package repository

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

const ScheduledMessageLimit = 100

// ScheduledMessageValidator checks current posting permissions. A failure reason
// makes the reservation failed; an error leaves it pending for a later retry.
type ScheduledMessageValidator func(context.Context, *model.ScheduledMessage) (recipients []uuid.UUID, failure string, err error)

type ScheduledMessageRepository interface {
	CreateScheduledMessage(ctx context.Context, m *model.ScheduledMessage) error
	GetScheduledMessages(ctx context.Context, userID uuid.UUID) ([]*model.ScheduledMessage, error)
	GetDueScheduledMessageIDs(ctx context.Context, now time.Time, limit int) ([]uuid.UUID, error)
	// CancelScheduledMessage locks against delivery. Sent reservations cannot be canceled.
	CancelScheduledMessage(ctx context.Context, id, userID uuid.UUID) (*model.ScheduledMessage, error)
	// ProcessScheduledMessage atomically publishes attachments, creates the message,
	// and marks it sent. Competing workers and cancellation serialize on the same row.
	ProcessScheduledMessage(ctx context.Context, id uuid.UUID, validate ScheduledMessageValidator) error
}
