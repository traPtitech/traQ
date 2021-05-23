package v3

import (
	"context"
	"net/http"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/leandro-lugaresi/hub"

	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/consts"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/router/utils"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/rbac/permission"
	"github.com/traPtitech/traQ/service/rbac/role"
	"github.com/traPtitech/traQ/utils/optional"
	"github.com/traPtitech/traQ/utils/validator"
)

// GetBots GET /bots
func (h *Handlers) GetBots(c echo.Context) error {
	var q repository.BotsQuery
	if !isTrue(c.QueryParam("all")) {
		q = q.CreatedBy(getRequestUserID(c))
	}

	list, err := h.Repo.GetBots(q)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, formatBots(list))
}

// PostBotRequest POST /bots リクエストボディ
type PostBotRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Endpoint    string `json:"endpoint"`
}

func (r PostBotRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.Name, validator.BotUserNameRuleRequired...),
		vd.Field(&r.DisplayName, vd.Required, vd.RuneLength(1, 32)),
		vd.Field(&r.Description, vd.Required, vd.RuneLength(0, 1000)),
		vd.Field(&r.Endpoint, vd.Required, is.URL, validator.NotInternalURL),
	)
}

// CreateBot POST /bots
func (h *Handlers) CreateBot(c echo.Context) error {
	var req PostBotRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	iconFileID, err := file.GenerateIconFile(h.FileManager, req.Name)
	if err != nil {
		return herror.InternalServerError(err)
	}

	b, err := h.Repo.CreateBot(req.Name, req.DisplayName, req.Description, iconFileID, getRequestUserID(c), req.Endpoint)
	if err != nil {
		switch {
		case err == repository.ErrAlreadyExists:
			return herror.Conflict("this name has already been used")
		default:
			return herror.InternalServerError(err)
		}
	}

	t, err := h.Repo.GetTokenByID(b.AccessTokenID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusCreated, formatBotDetail(b, t, make([]uuid.UUID, 0)))
}

// GetBot GET /bots/:botID
func (h *Handlers) GetBot(c echo.Context) error {
	b := getParamBot(c)

	if isTrue(c.QueryParam("detail")) {
		user := getRequestUser(c)

		// アクセス権確認
		if !h.RBAC.IsGranted(user.GetRole(), permission.AccessOthersBot) && b.CreatorID != user.GetID() {
			return herror.Forbidden()
		}

		t, err := h.Repo.GetTokenByID(b.AccessTokenID)
		if err != nil {
			switch err {
			case repository.ErrNotFound:
				return herror.HTTPError(http.StatusInternalServerError, "This bot's Access Token has been revoked unexpectedly. Please inform admin about this error.")
			default:
				return herror.InternalServerError(err)
			}
		}

		ids, err := h.Repo.GetParticipatingChannelIDsByBot(b.ID)
		if err != nil {
			return herror.InternalServerError(err)
		}

		return c.JSON(http.StatusOK, formatBotDetail(b, t, ids))
	}

	return c.JSON(http.StatusOK, formatBot(b))
}

// PatchBotRequest PATCH /bots/:botID リクエストボディ
type PatchBotRequest struct {
	DisplayName     optional.String     `json:"displayName"`
	Description     optional.String     `json:"description"`
	Endpoint        optional.String     `json:"endpoint"`
	Privileged      optional.Bool       `json:"privileged"`
	DeveloperID     optional.UUID       `json:"developerId"`
	SubscribeEvents model.BotEventTypes `json:"subscribeEvents"`
}

func (r PatchBotRequest) ValidateWithContext(ctx context.Context) error {
	return vd.ValidateStructWithContext(ctx, &r,
		vd.Field(&r.DisplayName, vd.RuneLength(1, 32)),
		vd.Field(&r.Description, vd.RuneLength(0, 1000)),
		vd.Field(&r.Endpoint, is.URL, validator.NotInternalURL),
		vd.Field(&r.DeveloperID, validator.NotNilUUID, utils.IsActiveHumanUserID),
		vd.Field(&r.SubscribeEvents, utils.IsValidBotEvents),
	)
}

// EditBot PATCH /bots/:botID
func (h *Handlers) EditBot(c echo.Context) error {
	b := getParamBot(c)

	var req PatchBotRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	if req.Privileged.Valid && getRequestUser(c).GetRole() != role.Admin {
		return herror.Forbidden("you are not permitted to set privileged flag to bots")
	}

	args := repository.UpdateBotArgs{
		DisplayName:     req.DisplayName,
		Description:     req.Description,
		WebhookURL:      req.Endpoint,
		Privileged:      req.Privileged,
		CreatorID:       req.DeveloperID,
		SubscribeEvents: req.SubscribeEvents,
	}

	if err := h.Repo.UpdateBot(b.ID, args); err != nil {
		switch {
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		default:
			return herror.InternalServerError(err)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteBot DELETE /bots/:botID
func (h *Handlers) DeleteBot(c echo.Context) error {
	b := getParamBot(c)

	if err := h.Repo.DeleteBot(b.ID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetBotIcon GET /bots/:botID/icon
func (h *Handlers) GetBotIcon(c echo.Context) error {
	w := getParamBot(c)

	// ユーザー取得
	user, err := h.Repo.GetUser(w.BotUserID, false)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return utils.ServeUserIcon(c, h.FileManager, user)
}

// ChangeBotIcon PUT /bots/:botID/icon
func (h *Handlers) ChangeBotIcon(c echo.Context) error {
	return utils.ChangeUserIcon(h.Imaging, c, h.Repo, h.FileManager, getParamBot(c).BotUserID)
}

// GetBotLogsRequest GET /bots/:botID/logs リクエストクエリ
type GetBotLogsRequest struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

func (r *GetBotLogsRequest) Validate() error {
	if r.Limit == 0 {
		r.Limit = 30
	}
	return vd.ValidateStruct(r,
		vd.Field(&r.Limit, vd.Min(1), vd.Max(200)),
		vd.Field(&r.Offset, vd.Min(0)),
	)
}

// GetBotLogs GET /bots/:botID/logs
func (h *Handlers) GetBotLogs(c echo.Context) error {
	b := getParamBot(c)

	var req GetBotLogsRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	logs, err := h.Repo.GetBotEventLogs(b.ID, req.Limit, req.Offset)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, formatBotEventLogs(logs))
}

// GetChannelBots GET /channels/:channelID/bots
func (h *Handlers) GetChannelBots(c echo.Context) error {
	channelID := getParamAsUUID(c, consts.ParamChannelID)

	bots, err := h.Repo.GetBots(repository.BotsQuery{}.CMemberOf(channelID))
	if err != nil {
		return herror.InternalServerError(err)
	}

	res := make([]echo.Map, len(bots))
	for i, v := range bots {
		res[i] = echo.Map{
			"id":        v.ID,
			"botUserId": v.BotUserID,
		}
	}
	return c.JSON(http.StatusOK, res)
}

// ActivateBot POST /bots/:botID/actions/activate
func (h *Handlers) ActivateBot(c echo.Context) error {
	b := getParamBot(c)

	h.Hub.Publish(hub.Message{
		Name: event.BotPingRequest,
		Fields: hub.Fields{
			"bot_id": b.ID,
			"bot":    b,
		},
	})
	return c.NoContent(http.StatusAccepted)
}

// InactivateBot POST /bots/:botID/actions/inactivate
func (h *Handlers) InactivateBot(c echo.Context) error {
	b := getParamBot(c)

	if err := h.Repo.ChangeBotState(b.ID, model.BotInactive); err != nil {
		return herror.InternalServerError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ReissueBot POST /bots/:botID/actions/reissue
func (h *Handlers) ReissueBot(c echo.Context) error {
	b := getParamBot(c)

	b, err := h.Repo.ReissueBotTokens(b.ID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	t, err := h.Repo.GetTokenByID(b.AccessTokenID)
	if err != nil {
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"verificationToken": b.VerificationToken,
		"accessToken":       t.AccessToken,
	})
}

// PostBotActionJoinRequest POST /bots/:botID/actions/join リクエストボディ
type PostBotActionJoinRequest struct {
	ChannelID uuid.UUID `json:"channelId"`
}

func (r PostBotActionJoinRequest) ValidateWithContext(ctx context.Context) error {
	return vd.ValidateStructWithContext(ctx, &r,
		vd.Field(&r.ChannelID, vd.Required, validator.NotNilUUID, utils.IsPublicChannelID), // 公開チャンネルのみ許可
	)
}

// LetBotJoinChannel POST /bots/:botID/actions/join
func (h *Handlers) LetBotJoinChannel(c echo.Context) error {
	var req PostBotActionJoinRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	b := getParamBot(c)

	// 参加
	if err := h.Repo.AddBotToChannel(b.ID, req.ChannelID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// PostBotActionLeaveRequest POST /bots/:botID/actions/leave リクエストボディ
type PostBotActionLeaveRequest struct {
	ChannelID uuid.UUID `json:"channelId"`
}

func (r PostBotActionLeaveRequest) Validate() error {
	return vd.ValidateStruct(&r,
		vd.Field(&r.ChannelID, vd.Required, validator.NotNilUUID),
	)
}

// LetBotLeaveChannel POST /bots/:botID/actions/leave
func (h *Handlers) LetBotLeaveChannel(c echo.Context) error {
	var req PostBotActionLeaveRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}

	b := getParamBot(c)

	// 退出
	if err := h.Repo.RemoveBotFromChannel(b.ID, req.ChannelID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}
