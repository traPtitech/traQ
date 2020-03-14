package middlewares

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac"
	"github.com/traPtitech/traQ/rbac/permission"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"net/http"
)

// AccessControlMiddlewareGenerator アクセスコントロールミドルウェアのジェネレーターを返します
func AccessControlMiddlewareGenerator(r rbac.RBAC) func(p ...rbac.Permission) echo.MiddlewareFunc {
	return func(p ...rbac.Permission) echo.MiddlewareFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				// OAuth2スコープ権限検証
				if scopes, ok := c.Get(consts.KeyOAuth2AccessScopes).(model.AccessScopes); ok {
					for _, v := range p {
						if !r.IsAnyGranted(scopes.StringArray(), v) {
							// NG
							return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
						}
					}
				}

				// ユーザー権限検証
				user := c.Get(consts.KeyUser).(*model.User)
				for _, v := range p {
					if !r.IsGranted(user.Role, v) {
						// NG
						return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
					}
				}

				return next(c) // OK
			}
		}
	}
}

// AdminOnly 管理者ユーザーのみを通すミドルウェア
func AdminOnly(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// ユーザーロール検証
		user := c.Get(consts.KeyUser).(*model.User)
		if user.Role != role.Admin {
			return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
		}
		return next(c) // OK
	}
}

// BlockBot Botのリクエストを制限するミドルウェア
func BlockBot(repo repository.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get(consts.KeyUser).(*model.User)
			if user.Bot {
				return herror.Forbidden("your bot is not permitted to access this API")
			}
			return next(c)
		}
	}
}

// CheckBotAccessPerm BOTアクセス権限を確認するミドルウェア
func CheckBotAccessPerm(rbac rbac.RBAC, repo repository.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get(consts.KeyUser).(*model.User)
			b := c.Get(consts.KeyParamBot).(*model.Bot)

			// アクセス権確認
			if !rbac.IsGranted(user.Role, permission.AccessOthersBot) && b.CreatorID != user.ID {
				return herror.Forbidden()
			}

			return next(c)
		}
	}
}

// CheckWebhookAccessPerm Webhookアクセス権限を確認するミドルウェア
func CheckWebhookAccessPerm(rbac rbac.RBAC, repo repository.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get(consts.KeyUser).(*model.User)
			w := c.Get(consts.KeyParamWebhook).(model.Webhook)

			// アクセス権確認
			if !rbac.IsGranted(user.Role, permission.AccessOthersWebhook) && w.GetCreatorID() != user.ID {
				return herror.Forbidden()
			}

			return next(c)
		}
	}
}

// CheckFileAccessPerm Fileアクセス権限を確認するミドルウェア
func CheckFileAccessPerm(rbac rbac.RBAC, repo repository.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := c.Get(consts.KeyUser).(*model.User).ID
			fileID := extension.GetRequestParamAsUUID(c, consts.ParamFileID)

			// アクセス権確認
			if ok, err := repo.IsFileAccessible(fileID, userID); err != nil {
				switch err {
				case repository.ErrNilID, repository.ErrNotFound:
					return herror.NotFound()
				default:
					return herror.InternalServerError(err)
				}
			} else if !ok {
				return herror.Forbidden()
			}

			return next(c)
		}
	}
}

// CheckClientAccessPerm Clientアクセス権限を確認するミドルウェア
func CheckClientAccessPerm(rbac rbac.RBAC, repo repository.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get(consts.KeyUser).(*model.User)
			oc := c.Get(consts.KeyParamClient).(*model.OAuth2Client)

			// アクセス権確認
			if !rbac.IsGranted(user.Role, permission.ManageOthersClient) && oc.CreatorID != user.ID {
				return herror.Forbidden()
			}

			return next(c)
		}
	}
}

// CheckMessageAccessPerm Messageアクセス権限を確認するミドルウェア
func CheckMessageAccessPerm(rbac rbac.RBAC, repo repository.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := c.Get(consts.KeyUser).(*model.User).ID
			m := c.Get(consts.KeyParamMessage).(*model.Message)

			// アクセス権確認
			if ok, err := repo.IsChannelAccessibleToUser(userID, m.ChannelID); err != nil {
				return herror.InternalServerError(err)
			} else if !ok {
				return herror.NotFound()
			}

			return next(c)
		}
	}
}

// CheckChannelAccessPerm Channelアクセス権限を確認するミドルウェア
func CheckChannelAccessPerm(rbac rbac.RBAC, repo repository.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := c.Get(consts.KeyUser).(*model.User).ID
			ch := c.Get(consts.KeyParamChannel).(*model.Channel)

			// アクセス権確認
			if ok, err := repo.IsChannelAccessibleToUser(userID, ch.ID); err != nil {
				return herror.InternalServerError(err)
			} else if !ok {
				return herror.NotFound()
			}

			return next(c)
		}
	}
}

// CheckUserGroupAdminPerm UserGroup管理者権限を確認するミドルウェア
func CheckUserGroupAdminPerm(rbac rbac.RBAC, repo repository.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := c.Get(consts.KeyUser).(*model.User).ID
			g := c.Get(consts.KeyParamGroup).(*model.UserGroup)

			if !g.IsAdmin(userID) {
				return herror.Forbidden()
			}

			return next(c)
		}
	}
}
