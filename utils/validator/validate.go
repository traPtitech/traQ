package validator

import (
	"errors"
	"github.com/traPtitech/traQ/utils/iputil"
	"net/url"
	"regexp"
	"strings"

	vd "github.com/go-ozzo/ozzo-validation/v4"
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

// isPrivateUseURIScheme schemeがネイティブアプリ向けのリバースドメイン形式のプライベートスキーム(例: com.example.app)かどうか
func isPrivateUseURIScheme(scheme string) bool {
	return privateUseURISchemeRegex.MatchString(strings.ToLower(scheme))
}

var errInvalidOAuth2RedirectURI = errors.New("must be a http(s) URL or a private-use URI scheme based on a reverse domain name")

// OAuth2RedirectURI OAuth2クライアントのリダイレクトURIとして有効
//
// ホストを持つhttp(s)のURLと、ネイティブアプリ向けのリバースドメイン形式のプライベートスキームのURI
// (例: com.example.app:/oauth2redirect) のみを許可します (RFC 8252 Section 7.1, 8.4)。
// 認可エンドポイントと同じく url.ParseRequestURI で解釈し、
// フラグメント (RFC 6749 Section 3.1.2) やユーザー情報を含むURIは許可しません。
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
		return errInvalidOAuth2RedirectURI
	}
	if len(s) == 0 {
		return nil
	}

	if strings.Contains(s, "#") {
		return errors.New("must not contain a fragment")
	}
	if strings.ContainsAny(s, " \t\r\n") {
		return errInvalidOAuth2RedirectURI
	}
	u, err := url.ParseRequestURI(s)
	if err != nil || u.User != nil {
		return errInvalidOAuth2RedirectURI
	}
	switch {
	case u.Scheme == "http" || u.Scheme == "https":
		// "https:1/path" のようなホストを持たないURIを弾く
		if len(u.Opaque) > 0 || len(u.Hostname()) == 0 {
			return errInvalidOAuth2RedirectURI
		}
	case isPrivateUseURIScheme(u.Scheme):
		if len(u.Host) == 0 && len(u.Path) == 0 && len(u.Opaque) == 0 {
			return errInvalidOAuth2RedirectURI
		}
	default:
		return errInvalidOAuth2RedirectURI
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
