package sessions

import (
	"github.com/hashicorp/golang-lru"
	"log"
	"sync"
	"time"
)

var sessions *cache

type cache struct {
	sync.Mutex
	sessions *lru.Cache
}

func init() {
	l, err := lru.NewWithEvict(cacheSize, func(key interface{}, value interface{}) {
		sess := value.(*Session)
		sess.RLock()
		defer sess.RUnlock()
		age := time.Since(sess.created)
		if age <= time.Duration(sessionMaxAge)*time.Second {
			store.Save(key.(string), sess)
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	sessions = &cache{
		sessions: l,
	}
}

func (c *cache) get(id string) (*Session, error) {
	c.Lock()
	defer c.Unlock()

	session, ok := c.sessions.Get(id)
	if !ok {
		var err error
		session, err = store.GetByID(id)
		if err != nil {
			return nil, err
		}

		if session != nil {
			c.sessions.Add(id, session)
		}
	}

	return session.(*Session), nil
}

func (c *cache) set(session *Session) error {
	c.Lock()
	defer c.Unlock()

	session.Lock()
	session.lastAccess = time.Now()
	id := session.id
	session.Unlock()

	c.sessions.Add(id, session)
	return store.Save(id, session)
}

func (c *cache) delete(id string) error {
	c.Lock()
	defer c.Unlock()

	c.sessions.Remove(id)

	return store.DestroyByID(id)
}

func (c *cache) purge() {
	c.Lock()
	defer c.Unlock()
	c.sessions.Purge()
}
