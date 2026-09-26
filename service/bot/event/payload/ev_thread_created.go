package payload

import (
	"time"

	"github.com/traPtitech/traQ/model"
)

// ThreadCreated THREAD_CREATEDイベントペイロード
type ThreadCreated struct {
	Base
	Thread Channel `json:"channel"`
}

func MakeThreadCreated(eventTime time.Time, th *model.Channel, parentChPath string, user model.UserInfo) *ThreadCreated {
	return &ThreadCreated{
		Base:   MakeBase(eventTime),
		Thread: MakeChannel(th, parentChPath+"/"+th.Name, user),
	}
}
