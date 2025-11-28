package v3

import (
	"context"
	"net/http"
	"sort"
	"time"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/samber/lo"
	"github.com/skip2/go-qrcode"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	"github.com/traPtitech/traQ/service/channel"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/oidc"
	"github.com/traPtitech/traQ/service/rbac/role"
	jwt2 "github.com/traPtitech/traQ/utils/jwt"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/validator"
)

// GetUsers GET /users
func (h *Handlers) GetUsers(c echo.Context) error {
	q := repository.UsersQuery{}

	if isTrue(c.QueryParam("include-suspended")) && len(c.QueryParam("name")) > 0 {
		return herror.BadRequest("include-suspended and name cannot be specified at the same time")
	}

	if len(c.QueryParam("name")) > 0 {
		q = q.NameOf(c.QueryParam("name"))
	} else if !isTrue(c.QueryParam("include-suspended")) {
		q = q.Active()
	}

	users, err := h.Repo.GetUsers(q)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return extension.ServeJSONWithETag(c, formatUsers(users))
}

// PostUserRequest POST /users リクエストボディ
type PostUserRequest struct {
	Name     string              `json:"name"`
	Password optional.Of[string] `json:"password"`
}

func (r PostUserRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, validator.UserNameRuleRequired...),
		vd.Field(&r.Password, append(validator.PasswordRule, validator.RequiredIfValid)...),
	)
}

// CreateUser POST /users
func (h *Handlers) CreateUser(c echo.Context) error {
	var req PostUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	iconFileID, err := file.GenerateIconFile(h.FileManager, req.Name)
	if err != nil {
		return herror.InternalServerError(err)
	}

	user, err := h.Repo.CreateUser(repository.CreateUserArgs{Name: req.Name, Password: req.Password.ValueOrZero(), Role: role.User, IconFileID: iconFileID})
	if err != nil {
		switch err {
		case repository.ErrAlreadyExists:
			return herror.Conflict("name conflicts")
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.JSON(http.StatusCreated, formatUserDetail(user, []model.UserTag{}, []uuid.UUID{}))
}

// GetMe GET /users/me
func (h *Handlers) GetMe(c echo.Context) error {
	me := getRequestUser(c)

	tags, err := h.Repo.GetUserTagsByUserID(me.GetID())
	if err != nil {
		return herror.InternalServerError(err)
	}

	groups, err := h.Repo.GetUserBelongingGroupIDs(me.GetID())
	if err != nil {
		return herror.InternalServerError(err)
	}
	return extension.ServeJSONWithETag(c, echo.Map{
		"id":          me.GetID(),
		"bio":         me.GetBio(),
		"groups":      groups,
		"tags":        formatUserTags(tags),
		"updatedAt":   me.GetUpdatedAt(),
		"lastOnline":  me.GetLastOnline(),
		"twitterId":   me.GetTwitterID(),
		"name":        me.GetName(),
		"displayName": me.GetResponseDisplayName(),
		"iconFileId":  me.GetIconFileID(),
		"bot":         me.IsBot(),
		"state":       me.GetState().Int(),
		"permissions": h.RBAC.GetGrantedPermissions(me.GetRole()),
		"homeChannel": me.GetHomeChannel(),
	})
}

type userAccessScopes struct{}

func (u userAccessScopes) Contains(_ model.AccessScope) bool {
	return true
}

// GetMeOIDC GET /users/me/oidc
func (h *Handlers) GetMeOIDC(c echo.Context) error {
	tokenScopes, ok := c.Get(consts.KeyOAuth2AccessScopes).(model.AccessScopes)
	scopes := lo.Ternary[oidc.ScopeChecker](ok, tokenScopes, userAccessScopes{})

	userInfo, err := h.OIDC.GetUserInfo(getRequestUserID(c), scopes)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusOK, userInfo)
}

// PatchMeRequest PATCH /users/me リクエストボディ
type PatchMeRequest struct {
	DisplayName optional.Of[string]    `json:"displayName"`
	TwitterID   optional.Of[string]    `json:"twitterId"`
	Bio         optional.Of[string]    `json:"bio"`
	HomeChannel optional.Of[uuid.UUID] `json:"homeChannel"`
}

func (r PatchMeRequest) ValidateWithContext(ctx context.Context) error {
	return vd.ValidateStructWithContext(ctx, &r,
		vd.Field(&r.DisplayName, vd.RuneLength(0, 32)),
		vd.Field(&r.TwitterID, validator.TwitterIDRule...),
		vd.Field(&r.Bio, vd.RuneLength(0, 1000)),
	)
}

// EditMe PATCH /users/me
func (h *Handlers) EditMe(c echo.Context) error {
	userID := getRequestUserID(c)

	var req PatchMeRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.HomeChannel.Valid {
		if req.HomeChannel.V != uuid.Nil {
			// チャンネル存在確認
			if !h.ChannelManager.PublicChannelTree().IsChannelPresent(req.HomeChannel.V) {
				return herror.BadRequest("invalid homeChannel")
			}
		}
	}

	args := repository.UpdateUserArgs{
		DisplayName: req.DisplayName,
		TwitterID:   req.TwitterID,
		Bio:         req.Bio,
		HomeChannel: req.HomeChannel,
	}
	if err := h.Repo.UpdateUser(userID, args); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// PutMyPasswordRequest PUT /users/me/password リクエストボディ
type PutMyPasswordRequest struct {
	Password    string `json:"password"`
	NewPassword string `json:"newPassword"`
}

func (r PutMyPasswordRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Password, vd.Required),
		vd.Field(&r.NewPassword, validator.PasswordRuleRequired...),
	)
}

// PutMyPassword PUT /users/me/password
func (h *Handlers) PutMyPassword(c echo.Context) error {
	var req PutMyPasswordRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	user := getRequestUser(c)

	// パスワード認証
	if err := user.Authenticate(req.Password); err != nil {
		return herror.Unauthorized("password is wrong")
	}

	return utils.ChangeUserPassword(c, h.Repo, h.SessStore, user.GetID(), req.NewPassword)
}

// GetMyQRCode GET /users/me/qr-code
func (h *Handlers) GetMyQRCode(c echo.Context) error {
	user := getRequestUser(c)

	// トークン生成
	now := time.Now()
	deadline := now.Add(5 * time.Minute)
	token, err := jwt2.Sign(jwt.MapClaims{
		"iat":         now.Unix(),
		"exp":         deadline.Unix(),
		"userId":      user.GetID(),
		"name":        user.GetName(),
		"displayName": user.GetDisplayName(),
	})
	if err != nil {
		return herror.InternalServerError(err)
	}

	if isTrue(c.QueryParam("token")) {
		// 画像じゃなくて生のトークンを返す
		return c.String(http.StatusOK, token)
	}

	// QRコード画像生成
	png, err := qrcode.Encode(token, qrcode.Low, 512)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.Blob(http.StatusOK, consts.MimeImagePNG, png)
}

// GetUserIcon GET /users/:userID/icon
func (h *Handlers) GetUserIcon(c echo.Context) error {
	return utils.ServeUserIcon(c, h.FileManager, getParamUser(c))
}

// ChangeUserIcon PUT /users/:userID/icon
func (h *Handlers) ChangeUserIcon(c echo.Context) error {
	return utils.ChangeUserIcon(h.Imaging, c, h.Repo, h.FileManager, getParamAsUUID(c, consts.ParamUserID))
}

// GetMyIcon GET /users/me/icon
func (h *Handlers) GetMyIcon(c echo.Context) error {
	return utils.ServeUserIcon(c, h.FileManager, getRequestUser(c))
}

// ChangeMyIcon PUT /users/me/icon
func (h *Handlers) ChangeMyIcon(c echo.Context) error {
	return utils.ChangeUserIcon(h.Imaging, c, h.Repo, h.FileManager, getRequestUserID(c))
}

// GetMyStampHistoryRequest GET /users/me/stamp-history リクエストクエリ
type GetMyStampHistoryRequest struct {
	Limit int `query:"limit"`
}

func (r *GetMyStampHistoryRequest) Validate() error {
	if r.Limit == 0 {
		r.Limit = 100
	}
	return vd.ValidateStruct(r,
		vd.Field(&r.Limit, vd.Min(1), vd.Max(100)),
	)
}

// GetMyStampHistory GET /users/me/stamp-history
func (h *Handlers) GetMyStampHistory(c echo.Context) error {
	var req GetMyStampHistoryRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	userID := getRequestUserID(c)
	history, err := h.Repo.GetUserStampHistory(userID, req.Limit)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, history)
}

// GetMyStampRecommendationsRequest GET /users/me/stamp-recommendations リクエストクエリ
type GetMyStampRecommendationsRequest struct {
	Limit int `query:"limit"`
}

func (r *GetMyStampRecommendationsRequest) Validate() error {
	if r.Limit == 0 {
		r.Limit = 100
	}
	return vd.ValidateStruct(r,
		vd.Field(&r.Limit, vd.Min(1), vd.Max(200)),
	)
}

// GetMyStampRecommendations GET /users/me/stamp-recommendations
func (h *Handlers) GetMyStampRecommendations(c echo.Context) error {
	var req GetMyStampRecommendationsRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	userID := getRequestUserID(c)
	recommendations, err := h.Repo.GetUserStampRecommendations(userID, req.Limit)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, recommendations)
}

// PostMyFCMDeviceRequest POST /users/me/fcm-device リクエストボディ
type PostMyFCMDeviceRequest struct {
	Token string `json:"token"`
}

func (r PostMyFCMDeviceRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Token, vd.Required, vd.RuneLength(1, 190)),
	)
}

// PostMyFCMDevice POST /users/me/fcm-device
func (h *Handlers) PostMyFCMDevice(c echo.Context) error {
	var req PostMyFCMDeviceRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	userID := getRequestUserID(c)
	if err := h.Repo.RegisterDevice(userID, req.Token); err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// PutUserPasswordRequest PUT /users/:userID/password リクエストボディ
type PutUserPasswordRequest struct {
	NewPassword string `json:"newPassword"`
}

func (r PutUserPasswordRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.NewPassword, validator.PasswordRuleRequired...),
	)
}

// ChangeUserPassword PUT /users/:userID/password
func (h *Handlers) ChangeUserPassword(c echo.Context) error {
	var req PutUserPasswordRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	return utils.ChangeUserPassword(c, h.Repo, h.SessStore, getParamAsUUID(c, consts.ParamUserID), req.NewPassword)
}

// GetUser GET /users/:userID
func (h *Handlers) GetUser(c echo.Context) error {
	user := getParamUser(c)

	tags, err := h.Repo.GetUserTagsByUserID(user.GetID())
	if err != nil {
		return herror.InternalServerError(err)
	}

	groups, err := h.Repo.GetUserBelongingGroupIDs(user.GetID())
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, formatUserDetail(user, tags, groups))
}

// PatchUserRequest PATCH /users/:userID リクエストボディ
type PatchUserRequest struct {
	DisplayName optional.Of[string] `json:"displayName"`
	TwitterID   optional.Of[string] `json:"twitterId"`
	Role        optional.Of[string] `json:"role"`
	State       optional.Of[int]    `json:"state"`
}

func (r PatchUserRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.DisplayName, vd.RuneLength(0, 32)),
		vd.Field(&r.TwitterID, validator.TwitterIDRule...),
		vd.Field(&r.Role, vd.RuneLength(0, 30)),
		vd.Field(&r.State, vd.Min(0), vd.Max(2)),
	)
}

// EditUser PATCH /users/:userID
func (h *Handlers) EditUser(c echo.Context) error {
	userID := getParamAsUUID(c, consts.ParamUserID)

	var req PatchUserRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	args := repository.UpdateUserArgs{
		DisplayName: req.DisplayName,
		TwitterID:   req.TwitterID,
		Role:        req.Role,
	}
	var deactivate bool
	if req.State.Valid {
		args.UserState = optional.From(model.UserAccountStatus(req.State.V))
		deactivate = req.State.V == model.UserAccountStatusDeactivated.Int()
	}

	if err := h.Repo.UpdateUser(userID, args); err != nil {
		return herror.InternalServerError(err)
	}
	// 凍結の際
	if deactivate {
		// 1. 有効なセッションを削除
		if err := h.SessStore.RevokeSessionsByUserID(userID); err != nil {
			return herror.InternalServerError(err)
		}
		// 2. 未読を削除（重いので一時的にコメントアウトしている; Hubのイベント経由などで非同期処理にすべき）
		// （DeleteUnreadsByUserIDなんてメソッドは無いので生やすこと）
		// 	if err := h.Repo.DeleteUnreadsByUserID(userID); err != nil {
		// 		return herror.InternalServerError(err)
		// 	}
	}

	return c.NoContent(http.StatusNoContent)
}

// GetMyChannelSubscriptions GET /users/me/subscriptions
func (h *Handlers) GetMyChannelSubscriptions(c echo.Context) error {
	subscriptions, err := h.Repo.GetChannelSubscriptions(repository.ChannelSubscriptionQuery{}.SetUser(getRequestUserID(c)))
	if err != nil {
		return herror.InternalServerError(err)
	}

	type response struct {
		ChannelID uuid.UUID `json:"channelId"`
		Level     int       `json:"level"`
	}
	result := make([]response, len(subscriptions))
	for i, subscription := range subscriptions {
		result[i] = response{ChannelID: subscription.ChannelID, Level: subscription.GetLevel().Int()}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ChannelID.String() < result[j].ChannelID.String() })

	return extension.ServeJSONWithETag(c, result)
}

// PutChannelSubscribeLevelRequest PUT /users/me/subscriptions/:channelID リクエストボディ
type PutChannelSubscribeLevelRequest struct {
	Level optional.Of[int] `json:"level"`
}

func (r PutChannelSubscribeLevelRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Level, vd.NotNil, vd.Min(0), vd.Max(2)),
	)
}

// SetChannelSubscribeLevel PUT /users/me/subscriptions/:channelID
func (h *Handlers) SetChannelSubscribeLevel(c echo.Context) error {
	channelID := getParamAsUUID(c, consts.ParamChannelID)

	var req PutChannelSubscribeLevelRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	ch, err := h.ChannelManager.GetChannel(channelID)
	if err != nil {
		if err == channel.ErrChannelNotFound {
			return herror.NotFound()
		}
		return herror.InternalServerError(err)
	}

	if err := h.ChannelManager.ChangeChannelSubscriptions(ch.ID, map[uuid.UUID]model.ChannelSubscribeLevel{getRequestUserID(c): model.ChannelSubscribeLevel(req.Level.V)}, false, getRequestUserID(c)); err != nil {
		switch err {
		case channel.ErrInvalidChannel:
			return herror.Forbidden("the channel's subscriptions is not configurable")
		case channel.ErrForcedNotification:
			return herror.Forbidden("the channel's subscriptions is not configurable")
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// GetUserStats GET /users/me/:userID/stats
func (h *Handlers) GetUserStats(c echo.Context) error {
	userID := getParamAsUUID(c, consts.ParamUserID)
	stats, err := h.Repo.GetUserStats(userID)
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusOK, stats)
}
