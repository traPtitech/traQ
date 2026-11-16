//go:generate mockgen -source=$GOFILE -destination=mock_$GOPACKAGE/mock_$GOFILE
package event

import (
	"sync"

	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/utils/jsonutil"
)

// Dispatcher Botイベント配送機
type Dispatcher interface {
	// Send Botにイベントを送信します
	Send(b *model.Bot, event model.BotEventType, body []byte) (ok bool)
}

// Unicast 単一のBOTにイベントを送信
func Unicast(d Dispatcher, ev model.BotEventType, payload interface{}, target *model.Bot) error {
	if target == nil {
		return nil
	}
	return Multicast(d, ev, payload, []*model.Bot{target})
}

// Multicast 複数のBOTにイベントを送信
func Multicast(d Dispatcher, ev model.BotEventType, payload interface{}, targets []*model.Bot) error {
	if len(targets) == 0 {
		return nil
	}
	buf, err := makePayloadJSON(&payload)
	if err != nil {
		return err
	}

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

func makePayloadJSON(payload interface{}) ([]byte, error) {
	b, err := jsonutil.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}
