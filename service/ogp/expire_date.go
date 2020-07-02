package ogp

import "time"

const CacheHours = 7 * 24

func GetCacheExpireDate() time.Time {
	return time.Now().Add(time.Duration(CacheHours) * time.Hour)
}
