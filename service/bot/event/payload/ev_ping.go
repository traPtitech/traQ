package payload

// Ping PINGイベントペイロード
type Ping struct {
	Base
}

func MakePing() *Ping {
	return &Ping{
		Base: MakeBase(),
	}
}
