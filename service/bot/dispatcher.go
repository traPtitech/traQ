package bot

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/service/bot/event"
	"sync"
)

type Dispatcher interface {
	Send(b *model.Bot, event event.Type, body []byte) (ok bool)
	Wait()
}

// Unicast 単一のBOTにイベントを送信
func Unicast(d Dispatcher, ev event.Type, payload interface{}, target *model.Bot) error {
	if target == nil {
		return nil
	}
	return Multicast(d, ev, payload, []*model.Bot{target})
}

// Multicast 複数のBOTにイベントを送信
func Multicast(d Dispatcher, ev event.Type, payload interface{}, targets []*model.Bot) error {
	if len(targets) == 0 {
		return nil
	}
	buf, release, err := makePayloadJSON(&payload)
	if err != nil {
		return fmt.Errorf("unexpected json encode error: %w", err)
	}
	defer release()

	var wg sync.WaitGroup
	done := make(map[uuid.UUID]bool, len(targets))
	for _, bot := range targets {
		if !done[bot.ID] {
			done[bot.ID] = true
			bot := bot
			wg.Add(1)
			go func() {
				defer wg.Done()
				d.Send(bot, ev, buf)
			}()
		}
	}
	wg.Wait()
	return nil
}
