package payload

import (
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"time"
)

// StampCreated STAMP_CREATEDイベントペイロード
type StampCreated struct {
	Base
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	FileID  uuid.UUID `json:"fileId"`
	Creator User      `json:"creator"`
}

func MakeStampCreated(et time.Time, stamp *model.Stamp, user model.UserInfo) *StampCreated {
	return &StampCreated{
		Base:    MakeBase(et),
		ID:      stamp.ID,
		Name:    stamp.Name,
		FileID:  stamp.FileID,
		Creator: MakeUser(user),
	}
}
