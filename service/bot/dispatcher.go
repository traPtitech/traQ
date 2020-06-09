package bot

import (
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
)

type Dispatcher interface {
	Send(b *model.Bot, event event.Type, body []byte) (ok bool)
	Wait()
}
