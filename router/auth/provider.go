package auth

import (
	"bytes"
	"github.com/disintegration/imaging"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/utils"
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
