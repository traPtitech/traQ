package oauth2

import (
	"github.com/satori/go.uuid"
	"sync"
)

// StoreMock テスト用のOAuth2ストア
type StoreMock struct {
	sync.RWMutex
	clients    map[string]*Client
	authorizes map[string]*AuthorizeData
	tokens     map[uuid.UUID]*Token
}

func NewStoreMock() *StoreMock {
	return &StoreMock{
		clients:    make(map[string]*Client),
		authorizes: make(map[string]*AuthorizeData),
		tokens:     make(map[uuid.UUID]*Token),
	}
}

func (s *StoreMock) GetClient(id string) (*Client, error) {
	s.RLock()
	defer s.RUnlock()
	oc, ok := s.clients[id]
	if !ok {
		return nil, ErrClientNotFound
	}

	return oc, nil
}

func (s *StoreMock) GetClientsByUser(userID uuid.UUID) (res []*Client, err error) {
	s.RLock()
	defer s.RUnlock()
	for _, v := range s.clients {
		if v.CreatorID == userID {
			res = append(res, v)
		}
	}
	return
}

func (s *StoreMock) SaveClient(client *Client) error {
	s.Lock()
	defer s.Unlock()
	s.clients[client.ID] = client
	return nil
}

func (s *StoreMock) UpdateClient(client *Client) error {
	s.Lock()
	defer s.Unlock()
	s.clients[client.ID] = client
	return nil
}

func (s *StoreMock) DeleteClient(id string) error {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.clients[id]; !ok {
		return ErrClientNotFound
	}

	delete(s.clients, id)
	return nil
}

func (s *StoreMock) SaveAuthorize(data *AuthorizeData) error {
	s.Lock()
	defer s.Unlock()
	s.authorizes[data.Code] = data
	return nil
}

func (s *StoreMock) GetAuthorize(code string) (*AuthorizeData, error) {
	s.RLock()
	defer s.RUnlock()
	oa, ok := s.authorizes[code]
	if !ok {
		return nil, ErrAuthorizeNotFound
	}

	return oa, nil
}

func (s *StoreMock) DeleteAuthorize(code string) error {
	s.Lock()
	defer s.Unlock()
	if _, ok := s.authorizes[code]; !ok {
		return ErrAuthorizeNotFound
	}

	delete(s.authorizes, code)
	return nil
}

func (s *StoreMock) SaveToken(token *Token) error {
	s.Lock()
	defer s.Unlock()
	s.tokens[token.ID] = token
	return nil
}

func (s *StoreMock) GetTokenByID(id uuid.UUID) (*Token, error) {
	s.RLock()
	defer s.RUnlock()
	t, ok := s.tokens[id]
	if !ok {
		return nil, ErrTokenNotFound
	}

	return t, nil
}

func (s *StoreMock) GetTokenByAccess(access string) (*Token, error) {
	s.RLock()
	defer s.RUnlock()
	for _, v := range s.tokens {
		if v.AccessToken == access {
			return v, nil
		}
	}
	return nil, ErrTokenNotFound
}

func (s *StoreMock) DeleteTokenByAccess(access string) error {
	s.Lock()
	defer s.Unlock()
	for _, v := range s.tokens {
		if v.AccessToken == access {
			delete(s.tokens, v.ID)
			return nil
		}
	}
	return ErrTokenNotFound
}

func (s *StoreMock) GetTokenByRefresh(refresh string) (*Token, error) {
	s.RLock()
	defer s.RUnlock()
	for _, v := range s.tokens {
		if v.RefreshToken == refresh {
			return v, nil
		}
	}
	return nil, ErrTokenNotFound
}

func (s *StoreMock) DeleteTokenByRefresh(refresh string) error {
	s.Lock()
	defer s.Unlock()
	for _, v := range s.tokens {
		if v.RefreshToken == refresh {
			delete(s.tokens, v.ID)
			return nil
		}
	}
	return ErrTokenNotFound
}

func (s *StoreMock) GetTokensByUser(userID uuid.UUID) (res []*Token, err error) {
	s.RLock()
	defer s.RUnlock()
	for _, v := range s.tokens {
		if v.UserID == userID {
			res = append(res, v)
		}
	}
	return
}

func (s *StoreMock) DeleteTokenByUser(userID uuid.UUID) error {
	s.Lock()
	defer s.Unlock()

	var target []*Token

	for _, v := range s.tokens {
		if v.UserID == userID {
			target = append(target, v)
		}
	}

	for _, v := range target {
		delete(s.tokens, v.ID)
	}

	return nil
}

func (s *StoreMock) DeleteTokenByClient(clientID string) error {
	s.Lock()
	defer s.Unlock()

	var target []*Token

	for _, v := range s.tokens {
		if v.ClientID == clientID {
			target = append(target, v)
		}
	}

	for _, v := range target {
		delete(s.tokens, v.ID)
	}

	return nil
}
