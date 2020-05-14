package middlewares

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/service/rbac/role"
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
				user := c.Get(consts.KeyUser).(model.UserInfo)
				for _, v := range p {
					if !r.IsGranted(user.GetRole(), v) {
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
		user := c.Get(consts.KeyUser).(model.UserInfo)
		if user.GetRole() != role.Admin {
			return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
		}
		return next(c) // OK
	}
}

// BlockBot Botのリクエストを制限するミドルウェア
func BlockBot(repo repository.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get(consts.KeyUser).(model.UserInfo)
			if user.IsBot() {
				return herror.Forbidden("Bot users are not permitted to access this API")
			}
			return next(c)
		}
	}
}

// CheckBotAccessPerm BOTアクセス権限を確認するミドルウェア
func CheckBotAccessPerm(rbac rbac.RBAC, repo repository.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get(consts.KeyUser).(model.UserInfo)
			b := c.Get(consts.KeyParamBot).(*model.Bot)

			// アクセス権確認
			if b.BotUserID == user.GetID() {
				return next(c) // Bot自身のアクセス
			}
			if b.CreatorID == user.GetID() {
				return next(c) // Bot管理人のアクセス
			}
			if rbac.IsGranted(user.GetRole(), permission.AccessOthersBot) {
				return next(c) // 特権使用のアクセス
			}

			return herror.Forbidden()
		}
	}
}

// CheckWebhookAccessPerm Webhookアクセス権限を確認するミドルウェア
func CheckWebhookAccessPerm(rbac rbac.RBAC, repo repository.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get(consts.KeyUser).(model.UserInfo)
			w := c.Get(consts.KeyParamWebhook).(model.Webhook)

			// アクセス権確認
			if !rbac.IsGranted(user.GetRole(), permission.AccessOthersWebhook) && w.GetCreatorID() != user.GetID() {
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
			file := c.Get(consts.KeyParamFile).(model.FileMeta)
			userID := c.Get(consts.KeyUser).(model.UserInfo).GetID()

			if t := file.GetFileType(); t == model.FileTypeIcon || t == model.FileTypeStamp {
				// スタンプ・アイコン画像の場合はスキップ
				return next(c)
			}

			// アクセス権確認
			if ok, err := repo.IsFileAccessible(file.GetID(), userID); err != nil {
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
			user := c.Get(consts.KeyUser).(model.UserInfo)
			oc := c.Get(consts.KeyParamClient).(*model.OAuth2Client)

			// アクセス権確認
			if !rbac.IsGranted(user.GetRole(), permission.ManageOthersClient) && oc.CreatorID != user.GetID() {
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
			userID := c.Get(consts.KeyUser).(model.UserInfo).GetID()
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
			userID := c.Get(consts.KeyUser).(model.UserInfo).GetID()
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
			userID := c.Get(consts.KeyUser).(model.UserInfo).GetID()
			g := c.Get(consts.KeyParamGroup).(*model.UserGroup)

			if !g.IsAdmin(userID) {
				return herror.Forbidden()
			}

			return next(c)
		}
	}
}

// CheckClipFolderAccessPerm ClipFolderアクセス権限を確認するミドルウェア
func CheckClipFolderAccessPerm(rbac rbac.RBAC, repo repository.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get(consts.KeyUser).(model.UserInfo)
			cf := c.Get(consts.KeyParamClipFolder).(*model.ClipFolder)
			if user.GetID() == cf.OwnerID {
				return next(c) // 所有者のアクセス
			}

			return herror.Forbidden()
		}
	}
}
