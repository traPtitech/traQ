package oauth2

const (
	errInvalidRequest          = "invalid_request"
	errUnauthorizedClient      = "unauthorized_client"
	errAccessDenied            = "access_denied"
	errUnsupportedResponseType = "unsupported_response_type"
	errInvalidScope            = "invalid_scope"
	errServerError             = "server_error"
	errInvalidClient           = "invalid_client"
	errInvalidGrant            = "invalid_grant"
	errUnsupportedGrantType    = "unsupported_grant_type"
	errLoginRequired           = "login_required"
	errConsentRequired         = "consent_required"
)

var (
	// ErrInvalidScope OAuth2エラー 不正なスコープです
	ErrInvalidScope = &errorResponse{ErrorType: errInvalidScope}
	// ErrClientNotFound OAuth2エラー クライアントが存在しません
	ErrClientNotFound = &errorResponse{ErrorType: errInvalidClient}
	// ErrAuthorizeNotFound OAuth2エラー 認可コードが存在しません
	ErrAuthorizeNotFound = &errorResponse{ErrorType: errInvalidGrant}
	// ErrTokenNotFound OAuth2エラー トークンが存在しません
	ErrTokenNotFound = &errorResponse{ErrorType: errInvalidGrant}
	// ErrUserIDOrPasswordWrong OAuth2エラー ユーザー認証に失敗しました
	ErrUserIDOrPasswordWrong = &errorResponse{ErrorType: errInvalidGrant}
)

type errorResponse struct {
	ErrorType        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

// Error : エラータイプを返します。
func (e *errorResponse) Error() string {
	return e.ErrorType
}
