package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/repository"
	"net/http"
	"strconv"
	"time"
)

// GetPublicUserIcon GET /public/icon/{username}
func (h *Handlers) GetPublicUserIcon(c echo.Context) error {
	username := c.Param("username")

	// ユーザー取得
	user, err := h.Repo.GetUserByName(username)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return notFound()
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	// ファイルメタ取得
	meta, err := h.Repo.GetFileMeta(user.Icon)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return notFound()
		default:
			return internalServerError(err, h.requestContextLogger(c))
		}
	}

	// ファイルオープン
	file, err := h.Repo.GetFS().OpenFileByKey(meta.GetKey())
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.Mime)
	c.Response().Header().Set(headerETag, strconv.Quote(meta.Hash))
	c.Response().Header().Set(headerCacheControl, "public, max-age=3600") // 1時間キャッシュ
	http.ServeContent(c.Response(), c.Request(), meta.Name, meta.CreatedAt, file)
	return nil
}

// GetPublicEmojiJSON GET /public/emoji.json
func (h *Handlers) GetPublicEmojiJSON(c echo.Context) error {
	setLastModified(c, h.emojiJSONTime)
	if done, _ := checkPreconditions(c, h.emojiJSONTime); done {
		return nil
	}

	// キャッシュ確認
	h.emojiJSONCacheLock.RLock()
	if h.emojiJSONCache.Len() > 0 {
		defer h.emojiJSONCacheLock.RUnlock()
		return c.JSONBlob(http.StatusOK, h.emojiJSONCache.Bytes())
	}
	h.emojiJSONCacheLock.RUnlock()

	// 生成
	h.emojiJSONCacheLock.Lock()
	defer h.emojiJSONCacheLock.Unlock()

	if h.emojiJSONCache.Len() > 0 { // リロード
		return c.JSONBlob(http.StatusOK, h.emojiJSONCache.Bytes())
	}

	if err := generateEmojiJSON(h.Repo, &h.emojiJSONCache); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	h.emojiJSONTime = time.Now()
	setLastModified(c, h.emojiJSONTime)
	return c.JSONBlob(http.StatusOK, h.emojiJSONCache.Bytes())
}

func generateEmojiJSON(repo repository.StampRepository, buf *bytes.Buffer) error {
	stamps, err := repo.GetAllStamps()
	if err != nil {
		return err
	}

	resData := make(map[string][]string)
	arr := make([]string, len(stamps))
	for i, stamp := range stamps {
		arr[i] = stamp.Name
	}
	resData["all"] = arr

	buf.Reset()
	return json.NewEncoder(buf).Encode(resData)
}

// GetPublicEmojiCSS GET /public/emoji.css
func (h *Handlers) GetPublicEmojiCSS(c echo.Context) error {
	setLastModified(c, h.emojiCSSTime)
	if done, _ := checkPreconditions(c, h.emojiCSSTime); done {
		return nil
	}

	// キャッシュ確認
	h.emojiCSSCacheLock.RLock()
	if h.emojiCSSCache.Len() > 0 {
		defer h.emojiCSSCacheLock.RUnlock()
		return c.Blob(http.StatusOK, "text/css", h.emojiCSSCache.Bytes())
	}
	h.emojiCSSCacheLock.RUnlock()

	// 生成
	h.emojiCSSCacheLock.Lock()
	defer h.emojiCSSCacheLock.Unlock()

	if h.emojiCSSCache.Len() > 0 { // リロード
		return c.Blob(http.StatusOK, "text/css", h.emojiCSSCache.Bytes())
	}

	if err := generateEmojiCSS(h.Repo, &h.emojiCSSCache); err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	h.emojiCSSTime = time.Now()
	setLastModified(c, h.emojiCSSTime)
	return c.Blob(http.StatusOK, "text/css", h.emojiCSSCache.Bytes())
}

func generateEmojiCSS(repo repository.StampRepository, buf *bytes.Buffer) error {
	stamps, err := repo.GetAllStamps()
	if err != nil {
		return err
	}

	buf.Reset()
	buf.WriteString(".emoji {display: inline-block; text-indent: 999%; white-space: nowrap; overflow: hidden; color: rgba(0, 0, 0, 0); background-size: contain}")
	buf.WriteString(".s24{width: 24px; height: 24px}")
	buf.WriteString(".s32{width: 32px; height: 32px}")
	for _, stamp := range stamps {
		buf.WriteString(fmt.Sprintf(".emoji.e_%s{background-image:url(/api/1.0/public/emoji/%s)}", stamp.Name, stamp.ID))
	}
	return nil
}

// GetPublicEmojiImage GET /public/emoji/{stampID}
func (h *Handlers) GetPublicEmojiImage(c echo.Context) error {
	s := getStampFromContext(c)

	meta, err := h.Repo.GetFileMeta(s.FileID)
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}

	file, err := h.Repo.GetFS().OpenFileByKey(meta.GetKey())
	if err != nil {
		return internalServerError(err, h.requestContextLogger(c))
	}
	defer file.Close()

	c.Response().Header().Set(echo.HeaderContentType, meta.Mime)
	c.Response().Header().Set(headerETag, strconv.Quote(meta.Hash))
	c.Response().Header().Set(headerCacheControl, "private, max-age=31536000") // 1年間キャッシュ
	http.ServeContent(c.Response(), c.Request(), meta.Name, meta.CreatedAt, file)
	return nil
}
