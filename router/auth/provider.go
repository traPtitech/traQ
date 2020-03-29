package auth

import (
	"bytes"
	"context"
	"github.com/disintegration/imaging"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/sessions"
	"github.com/traPtitech/traQ/utils"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"net/http"
	"time"
)

const (
	cookieName   = "traq_ext_auth_cookie"
	cookieMaxAge = 60 * 5
)

type Provider interface {
	FetchUserInfo(t *oauth2.Token) (UserInfo, error)
	LoginHandler(c echo.Context) error
	CallbackHandler(c echo.Context) error
	L() *zap.Logger
}

type UserInfo interface {
	GetProviderName() string
	GetID() string
	GetName() string
	GetDisplayName() string
	GetProfileImage() ([]byte, error)
	IsLoginAllowedUser() bool
}

func defaultLoginHandler(oac *oauth2.Config) echo.HandlerFunc {
	return func(c echo.Context) error {
		state := utils.RandAlphabetAndNumberString(32)
		c.SetCookie(&http.Cookie{
			Name:     cookieName,
			Value:    state,
			Path:     "/",
			Expires:  time.Now().Add(cookieMaxAge * time.Second),
			MaxAge:   cookieMaxAge,
			HttpOnly: true,
		})
		return c.Redirect(http.StatusFound, oac.AuthCodeURL(state))
	}
}

func defaultCallbackHandler(p Provider, oac *oauth2.Config, repo repository.Repository, allowSignUp bool) echo.HandlerFunc {
	return func(c echo.Context) error {
		code := c.QueryParam("code")
		state := c.QueryParam("state")
		if len(code) == 0 || len(state) == 0 {
			return herror.BadRequest("missing code or state")
		}

		cookie, err := c.Cookie(cookieName)
		if err != nil {
			return herror.BadRequest("missing cookie")
		}
		if cookie.Value != state {
			return herror.BadRequest("invalid state")
		}

		t, err := oac.Exchange(context.Background(), code)
		if err != nil {
			return herror.BadRequest("token exchange failed")
		}

		tu, err := p.FetchUserInfo(t)
		if err != nil {
			return herror.InternalServerError(err)
		}

		if !tu.IsLoginAllowedUser() {
			return c.String(http.StatusForbidden, "You are not permitted to access traQ")
		}

		user, err := repo.GetUserByExternalID(tu.GetProviderName(), tu.GetID(), false)
		if err != nil {
			if err != repository.ErrNotFound {
				return herror.InternalServerError(err)
			}

			if !allowSignUp {
				return herror.Unauthorized("You are not a member of traQ")
			}

			args := repository.CreateUserArgs{
				Name:        tu.GetName(),
				DisplayName: tu.GetDisplayName(),
				Role:        role.User,
				ExternalLogin: &model.ExternalProviderUser{
					ProviderName: tu.GetProviderName(),
					ExternalID:   tu.GetID(),
				},
			}

			if b, err := tu.GetProfileImage(); err == nil && b != nil {
				fid, err := processProfileIcon(repo, b)
				if err == nil {
					args.IconFileID = uuid.NullUUID{Valid: true, UUID: fid}
				}
			}

			user, err = repo.CreateUser(args)
			if err != nil {
				if err == repository.ErrAlreadyExists {
					return herror.Conflict("name conflicts") // TODO 名前被りをどうするか
				}
				return herror.InternalServerError(err)
			}
			p.L().Info("New user was created by external auth",
				zap.Stringer("id", user.GetID()),
				zap.String("name", user.GetName()),
				zap.String("providerName", tu.GetProviderName()),
				zap.String("externalId", tu.GetID()),
				zap.String("externalName", tu.GetName()))
		}

		// ユーザーのアカウント状態の確認
		if !user.IsActive() {
			return herror.Forbidden("this account is currently suspended")
		}

		sess, err := sessions.Get(c.Response(), c.Request(), true)
		if err != nil {
			return herror.InternalServerError(err)
		}

		if err := sess.SetUser(user.GetID()); err != nil {
			return herror.InternalServerError(err)
		}
		p.L().Info("User was logged in by external auth",
			zap.Stringer("id", user.GetID()),
			zap.String("name", user.GetName()),
			zap.String("providerName", tu.GetProviderName()),
			zap.String("externalId", tu.GetID()),
			zap.String("externalName", tu.GetName()))

		return c.Redirect(http.StatusFound, "/")
	}
}

func processProfileIcon(repo repository.Repository, src []byte) (uuid.UUID, error) {
	const maxImageSize = 256

	// デコード
	img, err := imaging.Decode(bytes.NewBuffer(src), imaging.AutoOrientation(true))
	if err != nil {
		return uuid.Nil, err
	}

	// リサイズ
	if size := img.Bounds().Size(); size.X > maxImageSize || size.Y > maxImageSize {
		img = imaging.Fit(img, maxImageSize, maxImageSize, imaging.Linear)
	}

	// PNGに戻す
	b := &bytes.Buffer{}
	_ = imaging.Encode(b, img, imaging.PNG)

	// ファイル保存
	f, err := repo.SaveFile(repository.SaveFileArgs{
		FileName: "icon",
		FileSize: int64(b.Len()),
		MimeType: consts.MimeImagePNG,
		FileType: model.FileTypeIcon,
		Src:      b,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return f.GetID(), nil
}
