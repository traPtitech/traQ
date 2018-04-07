package bot

// Dao Bot Dao
type Dao struct {
	store Store
}

// NewDao BotDaoを作成します
func NewDao(store Store) *Dao {
	return &Dao{
		store: store,
	}
}
