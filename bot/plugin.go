package bot

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/utils/validator"
	"net/url"
	"time"
)

type Plugin struct {
	ID                uuid.UUID
	BotUserID         uuid.UUID
	DisplayName       string `validate:"max=32,required"`
	Description       string `validate:"required"`
	Command           string `validate:"command,required"`
	Usage             string
	IconFileID        uuid.UUID
	VerificationToken string `validate:"required"`
	AccessTokenID     uuid.UUID
	PostURL           url.URL
	Activated         bool
	IsValid           bool
	CreatorID         uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Validate 構造体を検証します
func (p *Plugin) Validate() error {
	return validator.ValidateStruct(p)
}

func (h *Dao) CreatePlugin() {

}
