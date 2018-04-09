package bot

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/bot/events"
	"github.com/traPtitech/traQ/utils/validator"
	"math"
	"net/url"
	"time"
)

var (
	ErrBotNotFound         = errors.New("not found")
	ErrBotNotActivated     = errors.New("the bot is not activated")
	ErrBotActivationFailed = errors.New("activation failed")
)

// GeneralBot GeneralBot構造体
type GeneralBot struct {
	ID                uuid.UUID
	BotUserID         uuid.UUID
	Name              string `validate:"name,max=20,required"`
	DisplayName       string `validate:"max=32,required"`
	Description       string `validate:"required"`
	IconFileID        uuid.UUID
	VerificationToken string `validate:"required"`
	AccessTokenID     uuid.UUID
	PostURL           url.URL
	SubscribeEvents   []string
	Activated         bool
	IsValid           bool
	OwnerID           uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (b *GeneralBot) GetPostURL() string {
	return b.PostURL.String()
}

func (b *GeneralBot) GetVerificationToken() string {
	return b.VerificationToken
}

func (b *GeneralBot) GetBotUserID() uuid.UUID {
	return b.BotUserID
}

// Validate 構造体を検証します
func (b *GeneralBot) Validate() error {
	return validator.ValidateStruct(b)
}

type InstalledChannel struct {
	BotID       uuid.UUID
	ChannelID   uuid.UUID
	InstalledBy uuid.UUID
}

func (h *Dao) CreateBot(name, displayName, description string, ownerID, iconFileID uuid.UUID, postURL url.URL, subscribes []string) (GeneralBot, error) {
	b := &GeneralBot{
		ID:                uuid.NewV4(),
		BotUserID:         uuid.NewV4(),
		Name:              fmt.Sprintf("BOT_%s", name),
		DisplayName:       displayName,
		Description:       description,
		IconFileID:        iconFileID,
		VerificationToken: base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes()),
		PostURL:           postURL,
		SubscribeEvents:   subscribes,
		Activated:         false,
		IsValid:           true,
		OwnerID:           ownerID,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	t, err := h.oauth2.IssueAccessToken(nil, b.BotUserID, "", nil, math.MaxInt32, false) //TODO scope
	if err != nil {
		return GeneralBot{}, err
	}
	b.AccessTokenID = t.ID

	if err := b.Validate(); err != nil {
		return GeneralBot{}, err
	}
	if err := h.store.SaveGeneralBot(b); err != nil {
		return GeneralBot{}, err
	}
	h.generalBots.Store(b.ID, *b)
	return *b, nil
}

func (h *Dao) ReissueBotTokens(id uuid.UUID) (GeneralBot, error) {
	b, ok := h.GetBot(id)
	if !ok {
		return GeneralBot{}, ErrBotNotFound
	}

	if err := h.oauth2.DeleteTokenByID(b.AccessTokenID); err != nil {
		return GeneralBot{}, err
	}

	t, err := h.oauth2.IssueAccessToken(nil, b.BotUserID, "", nil, math.MaxInt32, false) //TODO scope
	if err != nil {
		return GeneralBot{}, err
	}
	b.AccessTokenID = t.ID
	b.VerificationToken = base64.RawURLEncoding.EncodeToString(uuid.NewV4().Bytes())

	if err := h.store.UpdateGeneralBot(&b); err != nil {
		return GeneralBot{}, err
	}
	h.generalBots.Store(b.ID, b)
	return b, nil
}

func (h *Dao) UpdateBot(b *GeneralBot) (err error) {
	b.UpdatedAt = time.Now()
	err = b.Validate()
	if err != nil {
		return err
	}

	if err := h.store.SaveGeneralBot(b); err != nil {
		return err
	}

	h.generalBots.Store(b.ID, *b)
	return nil
}

func (h *Dao) DeleteBot(id uuid.UUID) error {
	h.insChLock.Lock()
	defer h.insChLock.Unlock()
	b, ok := h.GetBot(id)
	if !ok {
		return ErrBotNotFound
	}

	if err := h.oauth2.DeleteTokenByID(b.AccessTokenID); err != nil {
		return err
	}

	// Botをチャンネルからアンインストール
	ics, ok := h.insChByID.Load(id)
	if ok {
		for _, v := range ics.(map[uuid.UUID]InstalledChannel) {
			h.store.UninstallBot(v.BotID, v.ChannelID)
			cc, _ := h.insChByChID.LoadOrStore(v.ChannelID, map[uuid.UUID]InstalledChannel{})
			delete(cc.(map[uuid.UUID]InstalledChannel), v.BotID)
		}
		h.insChByID.Delete(id)
	}

	b.Activated = false
	b.VerificationToken = ""
	b.AccessTokenID = uuid.Nil
	b.UpdatedAt = time.Now()

	if err := h.store.SaveGeneralBot(&b); err != nil {
		return err
	}

	h.generalBots.Store(b.ID, b)
	return nil
}

func (h *Dao) GetBot(id uuid.UUID) (GeneralBot, bool) {
	b, ok := h.generalBots.Load(id)
	if !ok {
		return GeneralBot{}, false
	}
	gb := b.(GeneralBot)
	if !gb.IsValid {
		return GeneralBot{}, false
	}
	return gb, true
}

func (h *Dao) GetAllBots() (arr []GeneralBot) {
	h.generalBots.Range(func(key, value interface{}) bool {
		v := value.(GeneralBot)
		if v.IsValid {
			arr = append(arr, v)
		}
		return true
	})
	return
}

func (h *Dao) GetInstalledChannels(botID uuid.UUID) (arr []InstalledChannel) {
	h.insChLock.RLock()
	defer h.insChLock.RUnlock()
	ics, ok := h.insChByID.Load(botID)
	if !ok {
		return
	}
	for _, v := range ics.(map[uuid.UUID]InstalledChannel) {
		arr = append(arr, v)
	}
	return
}

func (h *Dao) GetInstalledBots(channelID uuid.UUID) (arr []InstalledChannel) {
	h.insChLock.RLock()
	defer h.insChLock.RUnlock()
	ics, ok := h.insChByChID.Load(channelID)
	if !ok {
		return
	}
	for _, v := range ics.(map[uuid.UUID]InstalledChannel) {
		arr = append(arr, v)
	}
	return
}

func (h *Dao) InstallBot(botID, channelID, userID uuid.UUID) error {
	h.insChLock.Lock()
	defer h.insChLock.Unlock()
	b, ok := h.GetBot(botID)
	if !ok {
		return ErrBotNotFound
	}
	if !b.Activated {
		return ErrBotNotActivated
	}

	if err := h.store.InstallBot(botID, channelID, userID); err != nil {
		return err
	}

	ic := InstalledChannel{
		BotID:       botID,
		ChannelID:   channelID,
		InstalledBy: userID,
	}

	bc, _ := h.insChByID.LoadOrStore(botID, map[uuid.UUID]InstalledChannel{})
	bc.(map[uuid.UUID]InstalledChannel)[channelID] = ic
	cc, _ := h.insChByChID.LoadOrStore(channelID, map[uuid.UUID]InstalledChannel{})
	cc.(map[uuid.UUID]InstalledChannel)[botID] = ic

	return nil
}

func (h *Dao) UninstallBot(botID, channelID uuid.UUID) error {
	h.insChLock.Lock()
	defer h.insChLock.Unlock()
	_, ok := h.GetBot(botID)
	if !ok {
		return ErrBotNotFound
	}

	if err := h.store.UninstallBot(botID, channelID); err != nil {
		return err
	}

	bc, _ := h.insChByID.LoadOrStore(botID, map[uuid.UUID]InstalledChannel{})
	delete(bc.(map[uuid.UUID]InstalledChannel), channelID)
	cc, _ := h.insChByChID.LoadOrStore(channelID, map[uuid.UUID]InstalledChannel{})
	delete(cc.(map[uuid.UUID]InstalledChannel), botID)

	return nil
}

func (h *Dao) ActivateBot(id uuid.UUID) error {
	b, ok := h.GetBot(id)
	if !ok {
		return ErrBotNotFound
	}

	if _, ok := h.activationStarted.LoadOrStore(id, struct{}{}); ok {
		return errors.New("the bot activation process has already started")
	}
	defer h.activationStarted.Delete(id)

	if _, err := h.sendEventToBot(&b, events.TEST, ""); err != nil {
		return ErrBotActivationFailed
	}

	b.Activated = true
	if err := h.UpdateBot(&b); err != nil {
		return err
	}
	return nil
}
