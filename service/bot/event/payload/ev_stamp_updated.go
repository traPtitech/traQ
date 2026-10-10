package payload

import (
	"time"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// StampUpdated STAMP_UPDATEDイベントペイロード
type StampUpdated struct {
	Base
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	FileID  uuid.UUID `json:"fileId"`
	Creator User      `json:"creator"`
}

func MakeStampUpdated(et time.Time, stamp *model.Stamp, user model.UserInfo) *StampUpdated {
	return &StampUpdated{
		Base:    MakeBase(et),
		ID:      stamp.ID,
		Name:    stamp.Name,
		FileID:  stamp.FileID,
		Creator: MakeUser(user),
	}
}
