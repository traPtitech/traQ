package bot

import (
	"encoding/base64"
	"errors"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"math"
	"net/url"
	"time"
)

var (
	ErrBotNotFound     = errors.New("not found")
	ErrBotInvalid      = errors.New("the bot is invalid")
	ErrBotNotActivated = errors.New("the bot is not activated")
)

// GeneralBot GeneralBot構造体
type GeneralBot struct {
	ID                uuid.UUID
	BotUserID         uuid.UUID
	Name              string `validate:"name,max=16,required"`
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

type InstalledChannel struct {
	BotID       uuid.UUID
	ChannelID   uuid.UUID
	InstalledBy uuid.UUID
}

// Validate 構造体を検証します
func (g *GeneralBot) Validate() error {
	return validator.ValidateStruct(g)
}

func (h *Dao) CreateBot(name, displayName, description string, ownerID, iconFileID uuid.UUID, postURL url.URL, subscribes []string) (GeneralBot, error) {
	b := &GeneralBot{
		ID:                uuid.NewV4(),
		BotUserID:         uuid.NewV4(),
		Name:              name,
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

	return *b, nil
}

func (h *Dao) GetBot(id uuid.UUID) (GeneralBot, bool) {
	return h.store.GetGeneralBot(id)
}

func (h *Dao) GetAllBots() []GeneralBot {
	return h.store.GetAllGeneralBots()
}

func (h *Dao) GetInstalledChannels(botID uuid.UUID) []InstalledChannel {
	return h.store.GetInstalledChannels(botID)
}

func (h *Dao) GetInstalledBots(channelID uuid.UUID) []InstalledChannel {
	return h.store.GetInstalledBot(channelID)
}

func (h *Dao) InstallBot(botID, channelID, userID uuid.UUID) error {
	b, ok := h.store.GetGeneralBot(botID)
	if !ok {
		return ErrBotNotFound
	}
	if !b.IsValid {
		return ErrBotInvalid
	}
	if !b.Activated {
		return ErrBotNotActivated
	}

	return h.store.InstallBot(botID, channelID, userID)
}

func (h *Dao) UninstallBot(botID, channelID uuid.UUID) error {
	_, ok := h.store.GetGeneralBot(botID)
	if !ok {
		return ErrBotNotFound
	}

	return h.store.UninstallBot(botID, channelID)
}
