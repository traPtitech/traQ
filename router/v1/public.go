package v1

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/motoki317/sc"

	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/file"
)

// GetPublicUserIcon GET /public/icon/{username}
func (h *Handlers) GetPublicUserIcon(c echo.Context) error {
	username := c.Param("username")

	// ユーザー取得
	user, err := h.Repo.GetUserByName(username, false)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return herror.NotFound()
		default:
			return herror.InternalServerError(err)
		}
	}

	// ファイルメタ取得
	meta, err := h.FileManager.Get(user.GetIconFileID())
	if err != nil {
		switch err {
		case file.ErrNotFound:
			return herror.NotFound()
		default:
			return herror.InternalServerError(err)
		}
	}

	// ファイルオープン
	file, err := meta.Open()
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.GetMIMEType())
	c.Response().Header().Set(consts.HeaderETag, strconv.Quote(meta.GetMD5Hash()))
	c.Response().Header().Set(consts.HeaderCacheControl, "public, max-age=3600") // 1時間キャッシュ
	http.ServeContent(c.Response(), c.Request(), meta.GetFileName(), meta.GetCreatedAt(), file)
	return nil
}

// GetPublicEmojiJSON GET /public/emoji.json
func (h *Handlers) GetPublicEmojiJSON(c echo.Context) error {
	emojiJSON, err := h.EmojiCache.json.Get(context.Background(), struct{}{})
	if err != nil {
		return herror.InternalServerError(err)
	}
	return extension.ServeWithETag(c, echo.MIMEApplicationJSONCharsetUTF8, emojiJSON)
}

// GetPublicEmojiCSS GET /public/emoji.css
func (h *Handlers) GetPublicEmojiCSS(c echo.Context) error {
	emojiCSS, err := h.EmojiCache.css.Get(context.Background(), struct{}{})
	if err != nil {
		return herror.InternalServerError(err)
	}
	return extension.ServeWithETag(c, "text/css", emojiCSS)
}

// GetPublicEmojiImage GET /public/emoji/{stampID}
func (h *Handlers) GetPublicEmojiImage(c echo.Context) error {
	s := getStampFromContext(c)

	meta, err := h.FileManager.Get(s.FileID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	file, err := meta.Open()
	if err != nil {
		return herror.InternalServerError(err)
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.GetMIMEType())
	c.Response().Header().Set(consts.HeaderETag, strconv.Quote(meta.GetMD5Hash()))
	c.Response().Header().Set(consts.HeaderCacheControl, "private, max-age=31536000") // 1年間キャッシュ
	http.ServeContent(c.Response(), c.Request(), meta.GetFileName(), meta.GetCreatedAt(), file)
	return nil
}

type EmojiCache struct {
	json *sc.Cache[struct{}, []byte]
	css  *sc.Cache[struct{}, []byte]
}

func NewEmojiCache(repo repository.Repository) *EmojiCache {
	return &EmojiCache{
		json: sc.NewMust(emojiJSONGenerator(repo), 365*24*time.Hour, 365*24*time.Hour),
		css:  sc.NewMust(emojiCSSGenerator(repo), 365*24*time.Hour, 365*24*time.Hour),
	}
}

// Purge purges cache content.
func (c *EmojiCache) Purge() {
	c.json.Purge()
	c.css.Purge()
}

func emojiJSONGenerator(repo repository.Repository) func(_ context.Context, _ struct{}) ([]byte, error) {
	return func(_ context.Context, _ struct{}) ([]byte, error) {
		stamps, err := repo.GetAllStampsWithThumbnail(repository.StampTypeAll)
		if err != nil {
			return nil, err
		}

		stampNames := make([]string, len(stamps))
		for i, stamp := range stamps {
			stampNames[i] = stamp.Name
		}

		var buf bytes.Buffer
		err = json.NewEncoder(&buf).Encode(map[string][]string{
			"all": stampNames,
		})
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), err
	}
}

func emojiCSSGenerator(repo repository.Repository) func(_ context.Context, _ struct{}) ([]byte, error) {
	return func(_ context.Context, _ struct{}) ([]byte, error) {
		stamps, err := repo.GetAllStampsWithThumbnail(repository.StampTypeAll)
		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		buf.WriteString(".emoji{display:inline-block;text-indent:999%;white-space:nowrap;overflow:hidden;color:rgba(0,0,0,0);background-size:contain}")
		buf.WriteString(".s16{width:16px;height:16px}")
		buf.WriteString(".s24{width:24px;height:24px}")
		buf.WriteString(".s32{width:32px;height:32px}")
		for _, stamp := range stamps {
			buf.WriteString(fmt.Sprintf(".emoji.e_%s{background-image:url(/api/1.0/public/emoji/%s)}", stamp.Name, stamp.ID))
		}
		return buf.Bytes(), nil
	}
}
