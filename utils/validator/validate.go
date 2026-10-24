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
	// privateUseURISchemeRegex リバースドメイン形式のURIスキーム(例: com.example.app)の正規表現
	privateUseURISchemeRegex = regexp.MustCompile(`^[a-z][a-z0-9+-]*(\.[a-z0-9+-]+)+$`)
)

// rfc3986AbsoluteURIRegex RFC 3986 の absolute-URI の正規表現
var rfc3986AbsoluteURIRegex = func() *regexp.Regexp {
	const (
		unreserved = `[A-Za-z0-9\-._~]`
		pctEncoded = `%[0-9A-Fa-f]{2}`
		subDelims  = `[!$&'()*+,;=]`
		h16        = `[0-9A-Fa-f]{1,4}`
		decOctet   = `(?:25[0-5]|2[0-4][0-9]|1[0-9]{2}|[1-9][0-9]|[0-9])`
	)
	pchar := `(?:` + unreserved + `|` + pctEncoded + `|` + subDelims + `|[:@])`
	ipv4 := decOctet + `\.` + decOctet + `\.` + decOctet + `\.` + decOctet
	ls32 := `(?:` + h16 + `:` + h16 + `|` + ipv4 + `)`
	ipv6 := strings.Join([]string{
		`(?:` + h16 + `:){6}` + ls32,
		`::(?:` + h16 + `:){5}` + ls32,
		`(?:` + h16 + `)?::(?:` + h16 + `:){4}` + ls32,
		`(?:(?:` + h16 + `:){0,1}` + h16 + `)?::(?:` + h16 + `:){3}` + ls32,
		`(?:(?:` + h16 + `:){0,2}` + h16 + `)?::(?:` + h16 + `:){2}` + ls32,
		`(?:(?:` + h16 + `:){0,3}` + h16 + `)?::` + h16 + `:` + ls32,
		`(?:(?:` + h16 + `:){0,4}` + h16 + `)?::` + ls32,
		`(?:(?:` + h16 + `:){0,5}` + h16 + `)?::` + h16,
		`(?:(?:` + h16 + `:){0,6}` + h16 + `)?::`,
	}, "|")
	ipLiteral := `\[(?:` + ipv6 + `|v[0-9A-Fa-f]+\.(?:` + unreserved + `|` + subDelims + `|:)+)\]`
	regName := `(?:` + unreserved + `|` + pctEncoded + `|` + subDelims + `)*`
	userinfo := `(?:` + unreserved + `|` + pctEncoded + `|` + subDelims + `|:)*`
	authority := `(?:` + userinfo + `@)?(?:` + ipLiteral + `|` + ipv4 + `|` + regName + `)(?::[0-9]*)?`
	hierPart := `(?://` + authority + `(?:/` + pchar + `*)*` +
		`|/(?:` + pchar + `+(?:/` + pchar + `*)*)?` +
		`|` + pchar + `+(?:/` + pchar + `*)*` +
		`|)`
	query := `(?:` + pchar + `|[/?])*`
	return regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+\-.]*:` + hierPart + `(?:\?` + query + `)?$`)
}()

var errInvalidOAuth2RedirectURI = errors.New("must be a valid URL or a URI with a private-use URI scheme based on a reverse domain name")

// OAuth2RedirectURI URL、またはリバースドメイン形式のスキームのURI
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
	if len(s) == 0 || is.URL.Validate(s) == nil {
		return nil
	}

	scheme, _, _ := strings.Cut(s, ":")
	if !privateUseURISchemeRegex.MatchString(strings.ToLower(scheme)) || !rfc3986AbsoluteURIRegex.MatchString(s) {
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
		if len(v) == 0 {
			return nil
		}
		s = v
	case optional.Of[string]:
		if len(v.ValueOrZero()) == 0 {
			return nil
		}
		s = v.V
	}
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
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
