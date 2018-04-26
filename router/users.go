package router

import (
	"bytes"
	"context"
	"fmt"
	"github.com/traPtitech/traQ/external/imagemagick"
	"github.com/traPtitech/traQ/utils/thumb"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/gommon/log"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"
	"github.com/traPtitech/traQ/rbac/role"
)

const (
	iconMaxWidth  = 256
	iconMaxHeight = 256
)

// UserForResponse クライアントに返す形のユーザー構造体
type UserForResponse struct {
	UserID      string `json:"userId"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	IconID      string `json:"iconFileId"`
	Bot         bool   `json:"bot"`
	TwitterID   string `json:"twitterId"`
}

// UserDetailForResponse クライアントに返す形の詳細ユーザー構造体
type UserDetailForResponse struct {
	UserID      string            `json:"userId"`
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName"`
	IconID      string            `json:"iconFileId"`
	Bot         bool              `json:"bot"`
	TwitterID   string            `json:"twitterId"`
	TagList     []*TagForResponse `json:"tagList"`
}

type loginRequestBody struct {
	Name string `json:"name" form:"name"`
	Pass string `json:"pass" form:"pass"`
}

// PostLogin Post /login のハンドラ
func PostLogin(c echo.Context) error {
	requestBody := &loginRequestBody{}
	err := c.Bind(requestBody)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprint(err))
	}

	user := &model.User{
		Name: requestBody.Name,
	}
	err = user.Authorization(requestBody.Pass)
	if err != nil {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Sprint(err))
	}

	sess, err := session.Get("sessions", c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("an error occurrerd while getting session: %v", err))
	}

	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 14,
		HttpOnly: true,
	}

	sess.Values["userID"] = user.ID
	sess.Save(c.Request(), c.Response())
	return c.NoContent(http.StatusNoContent)
}

// PostLogout Post /logout のハンドラ
func PostLogout(c echo.Context) error {
	sess, err := session.Get("sessions", c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "an error occurred while getting session")
	}

	sess.Values["userID"] = nil
	sess.Save(c.Request(), c.Response())
	return c.NoContent(http.StatusNoContent)
}

// GetUsers GET /users のハンドラ
func GetUsers(c echo.Context) error {
	users, err := model.GetUsers()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Can't get Users")
	}

	res := make([]*UserForResponse, 0)
	for _, user := range users {
		res = append(res, formatUser(user))
	}
	return c.JSON(http.StatusOK, res)
}

// GetMe GET /users/me のハンドラ
func GetMe(c echo.Context) error {
	me := c.Get("user").(*model.User)
	return c.JSON(http.StatusOK, formatUser(me))
}

// GetUserByID /GET /users/{userID} のハンドラ
func GetUserByID(c echo.Context) error {
	userID := c.Param("userID")
	user, err := model.GetUser(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	tagList, err := model.GetUserTagsByUserID(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	userDetail, err := formatUserDetail(user, tagList)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, userDetail)
}

// GetUserIcon GET /users/{userID}/icon のハンドラ
func GetUserIcon(c echo.Context) error {
	userID := c.Param("userID")

	user, err := model.GetUser(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	if _, ok := c.QueryParams()["thumb"]; ok {
		return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon+"/thumbnail")
	}

	return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon)
}

// GetMyIcon GET /users/me/icon のハンドラ
func GetMyIcon(c echo.Context) error {
	user := c.Get("user").(*model.User)
	if _, ok := c.QueryParams()["thumb"]; ok {
		return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon+"/thumbnail")
	}
	return c.Redirect(http.StatusFound, "/api/1.0/files/"+user.Icon)
}

// PutMyIcon Post /users/me/icon のハンドラ
func PutMyIcon(c echo.Context) error {
	user := c.Get("user").(*model.User)

	// file確認
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// ファイルサイズ制限1MB
	if uploadedFile.Size > 1024*1024 {
		return echo.NewHTTPError(http.StatusBadRequest, "too big image file")
	}

	// ファイルタイプ確認・必要があればリサイズ
	b := &bytes.Buffer{}
	src, err := uploadedFile.Open()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	defer src.Close()
	switch uploadedFile.Header.Get(echo.HeaderContentType) {
	case "image/png":
		img, err := png.Decode(src)
		if err != nil {
			// 不正なpngである
			return echo.NewHTTPError(http.StatusBadRequest, "bad png file")
		}
		if img.Bounds().Size().X > iconMaxWidth || img.Bounds().Size().Y > iconMaxHeight {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
			defer cancel()
			img, err = thumb.Resize(ctx, img, iconMaxWidth, iconMaxHeight)
			if err != nil {
				switch err {
				case context.DeadlineExceeded:
					// リサイズタイムアウト
					return echo.NewHTTPError(http.StatusBadRequest, "bad png file (resize timeout)")
				default:
					// 予期しないエラー
					c.Logger().Error(err)
					return echo.NewHTTPError(http.StatusInternalServerError)
				}
			}
		}

		// bytesに戻す
		if b, err = thumb.EncodeToPNG(img); err != nil {
			// 予期しないエラー
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

	case "image/jpeg":
		img, err := jpeg.Decode(src)
		if err != nil {
			// 不正なjpgである
			return echo.NewHTTPError(http.StatusBadRequest, "bad jpg file")
		}
		if img.Bounds().Size().X > iconMaxWidth || img.Bounds().Size().Y > iconMaxHeight {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
			defer cancel()
			img, err = thumb.Resize(ctx, img, iconMaxWidth, iconMaxHeight)
			if err != nil {
				switch err {
				case context.DeadlineExceeded:
					// リサイズタイムアウト
					return echo.NewHTTPError(http.StatusBadRequest, "bad jpg file (resize timeout)")
				default:
					// 予期しないエラー
					c.Logger().Error(err)
					return echo.NewHTTPError(http.StatusInternalServerError)
				}
			}
		}

		// PNGに変換
		if b, err = thumb.EncodeToPNG(img); err != nil {
			// 予期しないエラー
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

	case "image/gif":
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) //10秒以内に終わらないファイルは無効
		defer cancel()
		b, err = imagemagick.ResizeAnimationGIF(ctx, src, iconMaxWidth, iconMaxHeight, false)
		if err != nil {
			switch err {
			case imagemagick.ErrUnavailable:
				// gifは一時的にサポートされていない
				return echo.NewHTTPError(http.StatusBadRequest, "gif file is temporarily unsupported")
			case imagemagick.ErrUnsupportedType:
				// 不正なgifである
				return echo.NewHTTPError(http.StatusBadRequest, "bad gif file")
			case context.DeadlineExceeded:
				// リサイズタイムアウト
				return echo.NewHTTPError(http.StatusBadRequest, "bad gif file (resize timeout)")
			default:
				// 予期しないエラー
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}

	case "image/svg+xml":
		// TODO svgバリデーション
		io.Copy(b, src)

	default:
		return echo.NewHTTPError(http.StatusBadRequest, "invalid image file")
	}

	// アイコン画像保存
	file := &model.File{
		Name:      uploadedFile.Filename,
		Size:      int64(b.Len()),
		CreatorID: user.ID,
	}
	if err := file.Create(b); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// アイコン変更
	if err := user.UpdateIconID(file.ID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go notification.Send(events.UserIconUpdated, events.UserEvent{ID: user.ID})
	return c.NoContent(http.StatusOK)
}

// PatchMe PUT /users/me
func PatchMe(c echo.Context) error {
	user := c.Get("user").(*model.User)

	req := struct {
		ExPassword  string `json:"exPassword"`
		DisplayName string `json:"displayName"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		TwitterID   string `json:"twitterId"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}

	if req.Email == "" && req.Password == "" {
		user.DisplayName = req.DisplayName
		if req.TwitterID != "" {
			user.TwitterID = req.TwitterID
		}
		if err := user.Update(); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update user. Please check the format of displayName")
		}
		return c.NoContent(http.StatusNoContent)
	}

	if err := user.Authorization(req.ExPassword); err != nil {
		return c.JSON(http.StatusUnauthorized, "Password is wrong")
	}

	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	if req.TwitterID != "" {
		user.TwitterID = req.TwitterID
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Password != "" {
		if err := user.SetPassword(req.Password); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update password")
		}
	}

	if err := user.Update(); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update user. Please check the format of email, password or displayName")
	}

	go notification.Send(events.UserUpdated, events.UserEvent{ID: user.ID})
	return c.NoContent(http.StatusNoContent)
}

// PostUsers Post /users のハンドラ
func PostUsers(c echo.Context) error {
	req := struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	newUser := &model.User{
		Name:  req.Name,
		Email: req.Email,
		Role:  role.User.ID(),
	}
	if err := newUser.SetPassword(req.Password); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	if err := newUser.Create(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	go notification.Send(events.UserJoined, events.UserEvent{ID: newUser.ID})
	return c.NoContent(http.StatusCreated)
}

func formatUser(user *model.User) *UserForResponse {
	res := &UserForResponse{
		UserID:      user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		IconID:      user.Icon,
		Bot:         user.Bot,
		TwitterID:   user.TwitterID,
	}
	if len(res.DisplayName) == 0 {
		res.DisplayName = res.Name
	}
	return res
}

func formatUserDetail(user *model.User, tagList []*model.UsersTag) (*UserDetailForResponse, error) {
	userDetail := &UserDetailForResponse{
		UserID:      user.ID,
		Name:        user.Name,
		DisplayName: user.DisplayName,
		IconID:      user.Icon,
		Bot:         user.Bot,
		TwitterID:   user.TwitterID,
	}
	if len(userDetail.DisplayName) == 0 {
		userDetail.DisplayName = userDetail.Name
	}

	for _, tag := range tagList {
		formattedTag, err := formatTag(tag)
		if err != nil {
			return nil, err
		}
		userDetail.TagList = append(userDetail.TagList, formattedTag)
	}
	return userDetail, nil
}

func validateUserID(userID string) (*model.User, error) {
	u, err := model.GetUser(userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return nil, echo.NewHTTPError(http.StatusNotFound, "This user dosen't exist")
		default:
			log.Errorf("failed to get usee: %v", err)
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get user")
		}
	}
	return u, nil
}
