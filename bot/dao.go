package bot

import "github.com/traPtitech/traQ/oauth2"

// Dao Bot Dao
type Dao struct {
	store  Store
	oauth2 *oauth2.Handler
}

// NewDao BotDaoを作成します
func NewDao(store Store, oauth2 *oauth2.Handler) *Dao {
	return &Dao{
		store:  store,
		oauth2: oauth2,
	}
}
