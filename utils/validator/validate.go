package validator

import (
	"errors"
	"github.com/traPtitech/traQ/utils/iputil"
	"net/url"
	"regexp"
	"strings"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/utils/optional"
)

var (
	// ChannelRegex チャンネル名正規表現
	ChannelRegex = regexp.MustCompile(`^[a-zA-Z0-9-_]{1,20}$`)
	// TwitterIDRegex ツイッターIDの正規表現
	TwitterIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,15}$`)
	// PKCERegex PKCE文字列の正規表現
	PKCERegex = regexp.MustCompile("^[a-zA-Z0-9~._-]{43,128}$")
	// UserRoleNameRegex ユーザーロール名の正規表現
	UserRoleNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,30}$`)
	// privateUseURISchemeRegex リバースドメイン形式のプライベートスキーム名の正規表現 (RFC 8252 Section 7.1, RFC 3986 Section 3.1)
	//
	// "."を含まないスキーム(javascript, data等)はマッチしない
	privateUseURISchemeRegex = regexp.MustCompile(`^[a-z][a-z0-9+-]*(\.[a-z0-9+-]+)+$`)
)

// IsPrivateUseURIScheme schemeがネイティブアプリ向けのリバースドメイン形式のプライベートスキーム(例: com.example.app)かどうか
func IsPrivateUseURIScheme(scheme string) bool {
	return privateUseURISchemeRegex.MatchString(strings.ToLower(scheme))
}

// OAuth2RedirectURI OAuth2クライアントのリダイレクトURIとして有効
//
// 通常のURLに加えて、ネイティブアプリ向けにリバースドメイン形式のプライベートスキームのURI
// (例: com.example.app:/oauth2redirect) を許可します (RFC 8252 Section 7.1, 8.4)。
// フラグメントを含むURIは許可しません (RFC 6749 Section 3.1.2)。
var OAuth2RedirectURI = vd.By(func(value interface{}) error {
	var s string
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		s = v
	case optional.Of[string]:
		s = v.ValueOrZero()
	default:
		return errors.New("must be a valid redirect uri")
	}
	if len(s) == 0 {
		return nil
	}

	if strings.Contains(s, "#") {
		return errors.New("must not contain a fragment")
	}
	if is.URL.Validate(s) == nil {
		return nil
	}
	u, err := url.Parse(s)
	if err != nil || !IsPrivateUseURIScheme(u.Scheme) || (len(u.Host) == 0 && len(u.Path) == 0 && len(u.Opaque) == 0) {
		return errors.New("must be a valid URL or a private-use URI scheme based on a reverse domain name")
	}
	return nil
})

// NotInternalURL 内部ネットワーク宛のURLでない
var NotInternalURL = vd.By(func(value interface{}) error {
	var s string
	switch v := value.(type) {
	case nil:
		return nil
	case string:
		if len(s) == 0 {
			return nil
		}
		s = v
	case optional.Of[string]:
		if len(v.ValueOrZero()) == 0 {
			return nil
		}
		s = v.V
	}
	u, _ := url.Parse(s)
	if iputil.IsPrivateHost(u.Hostname()) {
		return errors.New("must not be internal url")
	}
	return nil
})

// NotNilUUID uuid.Nilでない
var NotNilUUID = vd.By(func(value interface{}) error {
	switch u := value.(type) {
	case nil:
		return nil
	case uuid.UUID:
		if u == uuid.Nil {
			return errors.New("invalid uuid")
		}
	case optional.Of[uuid.UUID]:
		if u.Valid && u.V == uuid.Nil {
			return errors.New("invalid uuid")
		}
	case string:
		if v := uuid.FromStringOrNil(u); v == uuid.Nil {
			return errors.New("invalid uuid")
		}
	case []byte:
		if v := uuid.FromBytesOrNil(u); v == uuid.Nil {
			return errors.New("invalid uuid")
		}
	default:
		return errors.New("invalid uuid")
	}
	return nil
})
