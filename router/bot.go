package router

import (
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo"
	"github.com/leandro-lugaresi/hub"
	"github.com/traPtitech/traQ/bot"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/rbac/role"
	"github.com/traPtitech/traQ/repository"
	"go.uber.org/zap"
	"gopkg.in/guregu/null.v3"
	"net/http"
	"time"
)

type botResponse struct {
	BotID           uuid.UUID       `json:"botId"`
	BotUserID       uuid.UUID       `json:"botUserId"`
	Description     string          `json:"description"`
	SubscribeEvents model.BotEvents `json:"subscribeEvents"`
	State           model.BotState  `json:"state"`
	CreatorID       uuid.UUID       `json:"creatorId"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

type botDetailResponse struct {
	BotID            uuid.UUID       `json:"botId"`
	BotUserID        uuid.UUID       `json:"botUserId"`
	Description      string          `json:"description"`
	SubscribeEvents  model.BotEvents `json:"subscribeEvents"`
	State            model.BotState  `json:"state"`
	CreatorID        uuid.UUID       `json:"creatorId"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
	VerificationCode string          `json:"verificationCode"`
	AccessToken      string          `json:"accessToken"`
	PostURL          string          `json:"postUrl"`
	Privileged       bool            `json:"privileged"`
	BotCode          string          `json:"botCode"`
}

// GetBots GET /bots
func (h *Handlers) GetBots(c echo.Context) error {
	list, err := h.Repo.GetBotsByCreator(getRequestUserID(c))
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*botResponse, len(list))
	for i, b := range list {
		res[i] = &botResponse{
			BotID:           b.ID,
			BotUserID:       b.BotUserID,
			Description:     b.Description,
			SubscribeEvents: b.SubscribeEvents,
			State:           b.State,
			CreatorID:       b.CreatorID,
			CreatedAt:       b.CreatedAt,
			UpdatedAt:       b.UpdatedAt,
		}
	}

	return c.JSON(http.StatusOK, res)
}

// PostBots POST /bots
func (h *Handlers) PostBots(c echo.Context) error {
	var req struct {
		Name        string `json:"name" validate:"required,name,max=16"`
		DisplayName string `json:"displayName" validate:"max=32"`
		Description string `json:"description" validate:"required"`
		WebhookURL  string `json:"webhookUrl" validate:"required,url"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	b, err := h.Repo.CreateBot(req.Name, req.DisplayName, req.Description, getRequestUserID(c), req.WebhookURL)
	if err != nil {
		switch err {
		case repository.ErrAlreadyExists:
			return echo.NewHTTPError(http.StatusConflict, "this name has already been used.")
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	t, err := h.Repo.GetTokenByID(b.AccessTokenID)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, &botDetailResponse{
		BotID:            b.ID,
		BotUserID:        b.BotUserID,
		Description:      b.Description,
		SubscribeEvents:  b.SubscribeEvents,
		State:            b.State,
		CreatorID:        b.CreatorID,
		CreatedAt:        b.CreatedAt,
		UpdatedAt:        b.UpdatedAt,
		VerificationCode: b.VerificationToken,
		AccessToken:      t.AccessToken,
		PostURL:          b.PostURL,
		Privileged:       b.Privileged,
		BotCode:          b.BotCode,
	})
}

// GetBot GET /bots/:botID
func (h *Handlers) GetBot(c echo.Context) error {
	b := getBotFromContext(c)
	return c.JSON(http.StatusOK, &botResponse{
		BotID:           b.ID,
		BotUserID:       b.BotUserID,
		Description:     b.Description,
		SubscribeEvents: b.SubscribeEvents,
		State:           b.State,
		CreatorID:       b.CreatorID,
		CreatedAt:       b.CreatedAt,
		UpdatedAt:       b.UpdatedAt,
	})
}

// PatchBot PATCH /bots/:botID
func (h *Handlers) PatchBot(c echo.Context) error {
	b := getBotFromContext(c)

	if b.CreatorID != getRequestUserID(c) {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	var req struct {
		DisplayName null.String `json:"displayName" validate:"max=32"`
		Description null.String `json:"description"`
		Privileged  null.Bool   `json:"privileged"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if req.DisplayName.Valid && len(req.DisplayName.String) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "displayName is empty")
	}

	if req.Privileged.Valid {
		if getRequestUser(c).Role != role.Admin.ID() {
			return echo.NewHTTPError(http.StatusForbidden)
		}
	}

	args := repository.UpdateBotArgs{
		DisplayName: req.DisplayName,
		Description: req.Description,
		Privileged:  req.Privileged,
	}

	if err := h.Repo.UpdateBot(b.ID, args); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteBot DELETE /bots/:botID
func (h *Handlers) DeleteBot(c echo.Context) error {
	b := getBotFromContext(c)

	if b.CreatorID != getRequestUserID(c) {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	if err := h.Repo.DeleteBot(b.ID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetBotDetail GET /bots/:botID/detail
func (h *Handlers) GetBotDetail(c echo.Context) error {
	b := getBotFromContext(c)

	if b.CreatorID != getRequestUserID(c) {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	t, err := h.Repo.GetTokenByID(b.AccessTokenID)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			h.requestContextLogger(c).Error("Bot's Access Token has been revoked unexpectedly", zap.Stringer("botId", b.ID), zap.Stringer("tokenId", b.AccessTokenID))
			return echo.NewHTTPError(http.StatusInternalServerError, "This bot's Access Token has been revoked unexpectedly. Please inform admin about this error.")
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.JSON(http.StatusOK, &botDetailResponse{
		BotID:            b.ID,
		BotUserID:        b.BotUserID,
		Description:      b.Description,
		SubscribeEvents:  b.SubscribeEvents,
		State:            b.State,
		CreatorID:        b.CreatorID,
		CreatedAt:        b.CreatedAt,
		UpdatedAt:        b.UpdatedAt,
		VerificationCode: b.VerificationToken,
		AccessToken:      t.AccessToken,
		PostURL:          b.PostURL,
		Privileged:       b.Privileged,
		BotCode:          b.BotCode,
	})
}

// PutBotEvents PUT /bots/:botID/events
func (h *Handlers) PutBotEvents(c echo.Context) error {
	b := getBotFromContext(c)

	if b.CreatorID != getRequestUserID(c) {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	var req struct {
		Events []string `json:"events" validate:"dive,required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	events := model.BotEvents{}
	for _, v := range req.Events {
		if !bot.IsEvent(v) {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid event: %s", v))
		}
		events[model.BotEvent(v)] = true
	}

	if err := h.Repo.SetSubscribeEventsToBot(b.ID, events); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PutBotIcon PUT /bots/:botID/icon
func (h *Handlers) PutBotIcon(c echo.Context) error {
	b := getBotFromContext(c)

	if b.CreatorID != getRequestUserID(c) {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	// file確認
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	iconID, err := h.processMultipartFormIconUpload(c, uploadedFile)
	if err != nil {
		return err
	}

	// アイコン変更
	if err := h.Repo.ChangeUserIcon(b.BotUserID, iconID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PutBotState PUT /bots/:botID/state
func (h *Handlers) PutBotState(c echo.Context) error {
	b := getBotFromContext(c)

	if b.CreatorID != getRequestUserID(c) {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	var req struct {
		State string `json:"state" validate:"required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	switch req.State {
	case "active":
		h.Hub.Publish(hub.Message{
			Name: event.BotPingRequest,
			Fields: hub.Fields{
				"bot_id": b.ID,
			},
		})
		return c.NoContent(http.StatusAccepted)
	case "inactive":
		if err := h.Repo.ChangeBotState(b.ID, model.BotInactive); err != nil {
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		return c.NoContent(http.StatusNoContent)
	default:
		return echo.NewHTTPError(http.StatusBadRequest)
	}
}

// PostBotReissueTokens POST /bots/:botID/reissue
func (h *Handlers) PostBotReissueTokens(c echo.Context) error {
	b := getBotFromContext(c)

	if b.CreatorID != getRequestUserID(c) {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	b, err := h.Repo.ReissueBotTokens(b.ID)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	t, err := h.Repo.GetTokenByID(b.AccessTokenID)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"verificationCode": b.VerificationToken,
		"accessToken":      t.AccessToken,
		"botCode":          b.BotCode,
	})
}

// GetBotJoinChannels GET /bots/:botID/channels
func (h *Handlers) GetBotJoinChannels(c echo.Context) error {
	b := getBotFromContext(c)

	if b.CreatorID != getRequestUserID(c) {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	ids, err := h.Repo.GetParticipatingChannelIDsByBot(b.ID)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, ids)
}

// GetBotEventLogs GET /bots/:botID/events/logs
func (h *Handlers) GetBotEventLogs(c echo.Context) error {
	b := getBotFromContext(c)

	var req struct {
		Limit  int `query:"limit"  validate:"min=0,max=50"`
		Offset int `query:"offset" validate:"min=0"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if req.Limit == 0 {
		req.Limit = 50
	}

	if b.CreatorID != getRequestUserID(c) {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	logs, err := h.Repo.GetBotEventLogs(b.ID, req.Limit, req.Offset)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, logs)
}

// GetChannelBots GET /channels/:channelID/bots
func (h *Handlers) GetChannelBots(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, paramChannelID)

	bots, err := h.Repo.GetBotsByChannel(channelID)
	if err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	type response struct {
		BotID     uuid.UUID `json:"botId"`
		BotUserID uuid.UUID `json:"botUserId"`
	}
	res := make([]response, len(bots))
	for i, v := range bots {
		res[i] = response{
			BotID:     v.ID,
			BotUserID: v.BotUserID,
		}
	}

	return c.JSON(http.StatusOK, res)
}

// PostChannelBots POST /channels/:channelID/bots
func (h *Handlers) PostChannelBots(c echo.Context) error {
	ch := getChannelFromContext(c)
	if !ch.IsPublic {
		return echo.NewHTTPError(http.StatusForbidden)
	}

	var req struct {
		Code string `json:"code" validate:"required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	b, err := h.Repo.GetBotByCode(req.Code)
	if err != nil {
		switch err {
		case repository.ErrNotFound:
			return echo.NewHTTPError(http.StatusBadRequest, "bot not found")
		default:
			h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if err := h.Repo.AddBotToChannel(b.ID, ch.ID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, map[string]uuid.UUID{"botId": b.ID})
}

// DeleteChannelBot DELETE /channels/:channelID/bots/:botID
func (h *Handlers) DeleteChannelBot(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, paramChannelID)
	botID := getRequestParamAsUUID(c, paramBotID)

	if err := h.Repo.RemoveBotFromChannel(botID, channelID); err != nil {
		h.requestContextLogger(c).Error(unexpectedError, zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}
