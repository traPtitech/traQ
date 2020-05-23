package search

var nullE = &nullEngine{}

type nullEngine struct{}

func NewNullEngine() Engine {
	return nullE
}

func (n *nullEngine) Available() bool {
	return false
}

func (n *nullEngine) Close() error {
	return nil
}
