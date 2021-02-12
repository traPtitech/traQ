package search

var nullE = &nullEngine{}

type nullEngine struct{}

// NewNullEngine 常に利用不可な検索エンジンを返します
func NewNullEngine() Engine {
	return nullE
}

func (n *nullEngine) Do(*Query) (Result, error) {
	return nil, ErrServiceUnavailable
}

func (n *nullEngine) Available() bool {
	return false
}

func (n *nullEngine) Close() error {
	return nil
}
