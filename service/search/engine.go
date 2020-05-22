package search

type Engine interface {
	Available() bool
	Close() error
}
