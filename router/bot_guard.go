package router

import (
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

type botGuardFunc func(h *Handlers, bot *model.Bot, c echo.Context) (bool, error)

// BotGuard Botのリクエストを制限するミドルウェア. PrivilegedなBOTは制限されない
func (h *Handlers) BotGuard(f botGuardFunc) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := getRequestUser(c)
			if !user.Bot {
				return next(c)
			}

			b, err := h.Repo.GetBotByBotUserID(user.ID)
			if err != nil {
				return internalServerError(err, h.requestContextLogger(c))
			}

			if b.Privileged {
				return next(c)
			}

			ok, err := f(h, b, c)
			if err != nil {
				return internalServerError(err, h.requestContextLogger(c))
			}
			if !ok {
				return forbidden("your bot is not permitted to access this API")
			}

			return next(c)
		}
	}
}

// blockAlways 常にBOTのリクエストを拒否
func blockAlways(h *Handlers, bot *model.Bot, c echo.Context) (bool, error) {
	return true, nil
}

// blockUnlessSubscribingEvent BOTが指定したイベントを購読していない場合にリクエストを拒否
func blockUnlessSubscribingEvent(event model.BotEvent) botGuardFunc {
	return func(h *Handlers, bot *model.Bot, c echo.Context) (b bool, e error) {
		return bot.SubscribeEvents.Contains(event), nil
	}
}
