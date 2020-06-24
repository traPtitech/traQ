package payload

import "time"

// Ping PINGイベントペイロード
type Ping struct {
	Base
}

func MakePing(et time.Time) *Ping {
	return &Ping{
		Base: MakeBase(et),
	}
}
