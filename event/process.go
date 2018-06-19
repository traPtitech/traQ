package event

import (
	"github.com/labstack/gommon/log"
	"time"
)

// Listener サーバーイベントリスナーのインターフェイス
type Listener interface {
	Process(t Type, time time.Time, data interface{}) error
}

var listeners []Listener

// Emit イベントを発行します
func Emit(t Type, data interface{}) {
	dt := time.Now()
	for _, l := range listeners {
		go func(l Listener) {
			if err := l.Process(t, dt, data); err != nil {
				log.Error(err)
			}
		}(l)
	}
}

// AddListener イベントリスナーを追加します
func AddListener(l Listener) {
	listeners = append(listeners, l)
}
