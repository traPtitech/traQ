package middlewares

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/message"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/service/rbac/role"
)

// AccessControlMiddlewareGenerator アクセスコントロールミドルウェアのジェネレーターを返します
func AccessControlMiddlewareGenerator(r rbac.RBAC) func(p ...permission.Permission) echo.MiddlewareFunc {
	return func(p ...permission.Permission) echo.MiddlewareFunc {
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
				if user == nil {
					for _, v := range p {
						if !r.IsGranted(role.Client, v) {
							// NG
							return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("you are not permitted to request to '%s'", c.Request().URL.Path))
						}
					}

					return next(c) // OK
				}
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

// BlockBot Botのリクエストを制限するミドルウェア
func BlockBot() echo.MiddlewareFunc {
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

// BlockNonBot Bot以外のリクエストを制限するミドルウェア
func BlockNonBot() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get(consts.KeyUser).(model.UserInfo)
			if !user.IsBot() {
				return herror.Forbidden("Non-bot users are not permitted to access this API")
			}
			return next(c)
		}
	}
}

// CheckBotAccessPerm BOTアクセス権限を確認するミドルウェア
func CheckBotAccessPerm(rbac rbac.RBAC) echo.MiddlewareFunc {
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
func CheckWebhookAccessPerm(rbac rbac.RBAC) echo.MiddlewareFunc {
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
func CheckFileAccessPerm(fm file.Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			f := c.Get(consts.KeyParamFile).(model.File)
			userID := c.Get(consts.KeyUser).(model.UserInfo).GetID()

			if t := f.GetFileType(); t == model.FileTypeIcon || t == model.FileTypeStamp {
				// スタンプ・アイコン画像の場合はスキップ
				return next(c)
			}

			// アクセス権確認
			if ok, err := fm.Accessible(f.GetID(), userID); err != nil {
				return herror.InternalServerError(err)
			} else if !ok {
				return herror.Forbidden()
			}

			return next(c)
		}
	}
}

// CheckClientAccessPerm Clientアクセス権限を確認するミドルウェア
func CheckClientAccessPerm(rbac rbac.RBAC) echo.MiddlewareFunc {
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
func CheckMessageAccessPerm(cm channel.Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := c.Get(consts.KeyUser).(model.UserInfo).GetID()
			channelID := c.Get(consts.KeyParamMessage).(message.Message).GetChannelID()

			// アクセス権確認
			if ok, err := cm.IsChannelAccessibleToUser(userID, channelID); err != nil {
				return herror.InternalServerError(err)
			} else if !ok {
				return herror.NotFound()
			}

			return next(c)
		}
	}
}

// CheckChannelAccessPerm Channelアクセス権限を確認するミドルウェア
func CheckChannelAccessPerm(cm channel.Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userID := c.Get(consts.KeyUser).(model.UserInfo).GetID()
			ch := c.Get(consts.KeyParamChannel).(*model.Channel)

			// アクセス権確認
			if ok, err := cm.IsChannelAccessibleToUser(userID, ch.ID); err != nil {
				return herror.InternalServerError(err)
			} else if !ok {
				return herror.NotFound()
			}

			return next(c)
		}
	}
}

// CheckUserGroupAdminPerm UserGroup管理者権限を確認するミドルウェア
func CheckUserGroupAdminPerm(rbac rbac.RBAC) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := c.Get(consts.KeyUser).(model.UserInfo)
			g := c.Get(consts.KeyParamGroup).(*model.UserGroup)

			if !g.IsAdmin(user.GetID()) && !rbac.IsGranted(user.GetRole(), permission.AllUserGroupsAdmin) {
				return herror.Forbidden()
			}

			return next(c)
		}
	}
}

// CheckClipFolderAccessPerm ClipFolderアクセス権限を確認するミドルウェア
func CheckClipFolderAccessPerm() echo.MiddlewareFunc {
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
