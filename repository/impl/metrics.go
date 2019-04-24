package impl

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	initialized     = false
	messagesCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "traq",
		Name:      "messages_count_total",
	})
	channelsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "traq",
		Name:      "channels_count_total",
	})
)
