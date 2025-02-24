package middlewares

import (
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/traPtitech/traQ/router/extension/herror"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type accessLog struct {
	countInLastMinute     int
	countInLastPeak       int
	lastAccessedAt        time.Time
	lastPeakStartedAt     time.Time
	lastReservationResult bool
	logs                  []int
}

func newAccessLog() *accessLog {
	return &accessLog{
		logs: make([]int, 12), // 5秒ごとに区切る
	}
}

func (l *accessLog) Add(reservationResult bool) {
	now := time.Now()
	if !reservationResult && l.lastReservationResult {
		l.countInLastPeak = 1
		l.lastPeakStartedAt = now
	} else {
		l.countInLastPeak++
	}

	nowSec := now.Second()
	secFromLastAccess := int64(now.Sub(l.lastAccessedAt).Seconds())
	if 60 <= secFromLastAccess {
		l.countInLastMinute = 1
		clear(l.logs)
		l.logs[nowSec/5] = 1
	} else {
		lastSec := l.lastAccessedAt.Second()
		nowIdx, lastIdx := nowSec/5, lastSec/5
		if nowIdx == lastIdx {
			l.logs[nowIdx]++
			l.countInLastMinute++
		} else {
			if lastIdx < nowIdx {
				for i := lastIdx + 1; i < nowIdx; i++ {
					l.countInLastMinute -= l.logs[i]
					l.logs[i] = 0
				}
			} else {
				for i := lastIdx + 1; i < len(l.logs); i++ {
					l.countInLastMinute -= l.logs[i]
					l.logs[i] = 0
				}
				for i := 0; i < nowIdx; i++ {
					l.countInLastMinute -= l.logs[i]
					l.logs[i] = 0
				}
			}
			l.countInLastMinute -= l.logs[nowIdx] - 1
			l.logs[nowIdx] = 1
		}
	}

	l.lastReservationResult = reservationResult
	l.lastAccessedAt = now
}

func (l *accessLog) CountInLastMinute() int {
	return l.countInLastMinute
}

func (l *accessLog) CountInLastPeak() int {
	return l.countInLastPeak
}

func (l *accessLog) SecondsFromLastPeakStarted() float64 {
	return l.lastAccessedAt.Sub(l.lastPeakStartedAt).Seconds()
}

func (l *accessLog) LastReservationResult() bool {
	return l.lastReservationResult
}

type accessLoggerMemoryStore map[string]*accessLog

func (s *accessLoggerMemoryStore) Get(id string) *accessLog {
	l, ok := (*s)[id]
	if !ok {
		l = newAccessLog()
		(*s)[id] = l
	}
	return l
}

func (s *accessLoggerMemoryStore) ClearUnused(expiresIn time.Duration) {
	ids := []string{}
	now := time.Now()
	for id, l := range *s {
		if expiresIn < now.Sub(l.lastAccessedAt) {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		(*s)[id] = nil
	}
}

func RateLimiterWithLogging(rate rate.Limit, burst int, logger *zap.Logger) echo.MiddlewareFunc {
	var (
		rateLimit = middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      rate,
			Burst:     burst,
			ExpiresIn: 3 * time.Minute,
		})
		accessLogs = &accessLoggerMemoryStore{}
		mutex      = &sync.Mutex{}
	)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			ok, err := rateLimit.Allow(ip)
			if err != nil {
				return herror.InternalServerError(err)
			}

			mutex.Lock()
			l := accessLogs.Get(ip)
			l.Add(ok)
			if !ok {
				logger.Warn(
					"Exceeded rate limit.",
					zap.String("path", c.Path()),
					zap.String("ip", ip),
					zap.Int("rateInAMinute", l.CountInLastMinute()),
					zap.Int("countInLastPeak", l.countInLastPeak),
					zap.Float64("rateInAPeak", float64(60*l.CountInLastPeak())/l.SecondsFromLastPeakStarted()),
				)
			}
			mutex.Unlock()
			return next(c)
		}
	}
}
