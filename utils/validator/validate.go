package validator

import (
	"errors"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/traPtitech/traQ/bot"
	"github.com/traPtitech/traQ/utils"
	"net/url"
	"regexp"
)

var (
	// ChannelRegex チャンネル名正規表現
	ChannelRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]{1,20}$`)
	// PasswordRegex パスワード正規表現
	PasswordRegex = regexp.MustCompile(`^[\x20-\x7E]{10,32}$`)
	// TwitterIDRegex ツイッターIDの正規表現
	TwitterIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,15}$`)
	// PKCERegex PKCE文字列の正規表現
	PKCERegex = regexp.MustCompile("^[a-zA-Z0-9~._-]{43,128}$")
)

// NotInternalURL 内部ネットワーク宛のURLでない
var NotInternalURL = validation.By(func(value interface{}) error {
	s, _ := value.(string)
	u, _ := url.Parse(s)
	if utils.IsPrivateHost(u.Hostname()) {
		return errors.New("must not be internal url")
	}
	return nil
})

// IsBotEvent BOTイベントかどうか
var IsBotEvent = validation.By(func(value interface{}) error {
	s, _ := value.(string)
	if bot.IsEvent(s) {
		return errors.New("must be bot event")
	}
	return nil
})
