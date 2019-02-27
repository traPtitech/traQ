package router

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

// GetPublicUserIcon GET /public/icon/{username}
func (h *Handlers) GetPublicUserIcon(c echo.Context) error {
	username := c.Param("username")
	if len(username) == 0 {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	// ユーザー取得
	user, err := h.Repo.GetUserByName(username)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	// ファイルオープン
	if _, ok := c.QueryParams()["thumb"]; ok {
		_, r, err := h.Repo.OpenThumbnailFile(user.Icon)
		if err != nil {
			switch err {
			case repository.ErrNotFound:
				return echo.NewHTTPError(http.StatusNotFound)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
		defer r.Close()
		c.Response().Header().Set(headerCacheControl, "public, max-age=3600") //1時間キャッシュ
		return c.Stream(http.StatusOK, mimeImagePNG, r)
	}

	f, r, err := h.Repo.OpenFile(user.Icon)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	defer r.Close()
	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(f.Size, 10))
	c.Response().Header().Set(headerCacheControl, "public, max-age=3600") //1時間キャッシュ
	return c.Stream(http.StatusOK, f.Mime, r)
}

// GetPublicEmojiJson GET /public/emoji.json
func (h *Handlers) GetPublicEmojiJson(c echo.Context) error {
	stamps, err := h.Repo.GetAllStamps()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	resData := make(map[string][]string)
	resData["all"] = make([]string, 0, len(stamps))
	for _, stamp := range stamps {
		resData["all"] = append(resData["all"], stamp.Name)
	}
	return c.JSON(http.StatusOK, resData)
}

// GetPublicEmojiCss GET /public/emoji.css
func (h *Handlers) GetPublicEmojiCss(c echo.Context) error {
	stamps, err := h.Repo.GetAllStamps()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	res := bytes.Buffer{}

	for _, stamp := range stamps {
		res.WriteString(fmt.Sprintf(".emoji.%s{background-image:url(/api/1.0/public/emoji/%s)}", stamp.Name, stamp.FileID))
	}
	return c.Blob(http.StatusOK, "text/css", res.Bytes())
}

// GetPublicEmojiImage GET /public/emoji/{fileID}
func (h *Handlers) GetPublicEmojiImage(c echo.Context) error {
	fileID := getRequestParamAsUUID(c, paramFileID)
	meta, err := h.Repo.GetFileMeta(fileID)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusBadRequest)
	}

	c.Response().Header().Set(headerFileMetaType, meta.Type)
	if meta.Type != model.FileTypeStamp {
		return c.NoContent(http.StatusForbidden)
	}

	c.Response().Header().Set(headerCacheFile, "true")

	// 直接アクセスURLが発行できる場合は、そっちにリダイレクト
	url, _ := h.Repo.GetFS().GenerateAccessURL(meta.GetKey())
	if len(url) > 0 {
		return c.Redirect(http.StatusFound, url)
	}

	file, err := h.Repo.GetFS().OpenFileByKey(meta.GetKey())
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.Mime)
	c.Response().Header().Set(echo.HeaderContentLength, strconv.FormatInt(meta.Size, 10))
	c.Response().Header().Set(headerCacheControl, "private, max-age=31536000") //1年間キャッシュ

	http.ServeContent(c.Response(), c.Request(), meta.Name, meta.CreatedAt, file)
	return nil
}
