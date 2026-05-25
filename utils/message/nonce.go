package message

import (
	"github.com/traPtitech/traQ/utils/set"
)

type NonceManager struct {
	nonceSet set.String
}

// NewNonceManager creates a new NonceManager with initialized set
func NewNonceManager() *NonceManager {
	return &NonceManager{
		nonceSet: set.String{},
	}
}

func (m *NonceManager) NonceChecker(nonce string) bool {
	if m.nonceSet.Contains(nonce) {
		return false
	}
	m.nonceSet.Add(nonce)
	return true
}
