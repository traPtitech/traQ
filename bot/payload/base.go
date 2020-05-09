package payload

import "time"

// Base 全イベントに埋め込まれるペイロード
type Base struct {
	EventTime time.Time `json:"eventTime"`
}

func MakeBase() Base {
	return Base{
		EventTime: time.Now(),
	}
}
