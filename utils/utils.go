package utils

func Map[T, R any](s []T, mapper func(item T) R) []R {
	ret := make([]R, len(s))
	for i := range s {
		ret[i] = mapper(s[i])
	}
	return ret
}
