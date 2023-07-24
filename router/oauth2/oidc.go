package oauth2

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/zitadel/oidc/pkg/oidc"

	"github.com/traPtitech/traQ/utils"
	"github.com/traPtitech/traQ/utils/jwt"
)

// OIDCDiscovery OpenID Connect Discovery のハンドラ
func (h *Handler) OIDCDiscovery(c echo.Context) error {
	return c.JSON(http.StatusOK, &oidc.DiscoveryConfiguration{
		Issuer:                           h.Origin,
		AuthorizationEndpoint:            h.Origin + "/api/v3/oauth2/authorize",
		TokenEndpoint:                    h.Origin + "/api/v3/oauth2/token",
		UserinfoEndpoint:                 h.Origin + "/api/v3/users/me/oidc",
		RevocationEndpoint:               h.Origin + "/api/v3/oauth2/revoke",
		EndSessionEndpoint:               h.Origin + "/api/v3/logout",
		JwksURI:                          h.Origin + "/api/v3/jwks",
		ScopesSupported:                  supportedScopes,
		ResponseTypesSupported:           supportedResponseTypes,
		ResponseModesSupported:           supportedResponseModes,
		GrantTypesSupported:              utils.Map(supportedGrantTypes, func(s string) oidc.GrantType { return oidc.GrantType(s) }),
		SubjectTypesSupported:            []string{"public"},
		IDTokenSigningAlgValuesSupported: jwt.SupportedAlgorithms(),
		CodeChallengeMethodsSupported:    utils.Map(supportedCodeChallengeMethods, func(s string) oidc.CodeChallengeMethod { return oidc.CodeChallengeMethod(s) }),
	})
}
