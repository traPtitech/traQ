package bot

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/oauth2"
	"net/http"
	"sync"
	"time"
)

// Dao Bot Dao
type Dao struct {
	store  Store
	oauth2 *oauth2.Handler

	webhooks    sync.Map
	plugins     sync.Map
	generalBots sync.Map

	insChByID   sync.Map
	insChByChID sync.Map
	insChLock   sync.RWMutex

	activationStarted sync.Map

	botReqClient http.Client
}

// NewDao BotDaoを作成します
func NewDao(store Store, oauth2 *oauth2.Handler) *Dao {
	dao := &Dao{
		store:  store,
		oauth2: oauth2,
		botReqClient: http.Client{
			Timeout: 5 * time.Second,
		},
	}

	if err := dao.init(); err != nil {
		panic(err)
	}

	return dao
}

func (h *Dao) init() error {
	webhooks, err := h.store.GetAllWebhooks()
	if err != nil {
		return err
	}
	for _, v := range webhooks {
		h.webhooks.Store(v.ID, v)
	}

	bots, err := h.store.GetAllGeneralBots()
	if err != nil {
		return err
	}
	for _, v := range bots {
		h.generalBots.Store(v.ID, v)
	}

	bics, err := h.store.GetAllBotsInstalledChannels()
	if err != nil {
		return err
	}
	for _, v := range bics {
		b, _ := h.insChByID.LoadOrStore(v.BotID, map[uuid.UUID]InstalledChannel{})
		b.(map[uuid.UUID]InstalledChannel)[v.ChannelID] = v
		c, _ := h.insChByChID.LoadOrStore(v.ChannelID, map[uuid.UUID]InstalledChannel{})
		c.(map[uuid.UUID]InstalledChannel)[v.BotID] = v
	}

	return nil
}
