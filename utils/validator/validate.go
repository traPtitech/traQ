package validator

import (
	"errors"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/gofrs/uuid"
	"github.com/traPtitech/traQ/utils"
	"net/url"
	"regexp"
)

var (
	// ChannelRegex チャンネル名正規表現
	ChannelRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]{1,20}$`)
	// TwitterIDRegex ツイッターIDの正規表現
	TwitterIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,15}$`)
	// PKCERegex PKCE文字列の正規表現
	PKCERegex = regexp.MustCompile("^[a-zA-Z0-9~._-]{43,128}$")
)

// NotInternalURL 内部ネットワーク宛のURLでない
var NotInternalURL = validation.By(func(value interface{}) error {
	s, _ := value.(string)
	if len(s) == 0 {
		return nil
	}
	u, _ := url.Parse(s)
	if utils.IsPrivateHost(u.Hostname()) {
		return errors.New("must not be internal url")
	}
	return nil
})

// NotNilUUID uuid.Nilでない
var NotNilUUID = validation.By(func(value interface{}) error {
	switch u := value.(type) {
	case uuid.UUID:
		if u == uuid.Nil {
			return errors.New("invalid uuid")
		}
	case string:
		if v := uuid.FromStringOrNil(u); v == uuid.Nil {
			return errors.New("invalid uuid")
		}
	}
	return nil
})
