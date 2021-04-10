package viewer

import "github.com/gofrs/uuid"

// StateWithChannel 閲覧状態
type StateWithChannel struct {
	State     State
	ChannelID uuid.UUID
}
