package router

import (
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/oauth2"
	"gopkg.in/go-playground/validator.v9"
	"net/url"
)

type botForResponse struct {
	BotID           string    `json:"botId"`
	BotUserID       string    `json:"botUserId"`
	Name            string    `json:"name"`
	DisplayName     string    `json:"displayName"`
	Description     string    `json:"description"`
	SubscribeEvents []string  `json:"subscribeEvents"`
	Activated       bool      `json:"activated"`
	CreatorID       string    `json:"creatorId"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type installedBotForResponse struct {
	BotID       string    `json:"botId"`
	InstalledBy string    `json:"installedBy"`
	InstalledAt time.Time `json:"installedAt"`
}

// GetBots GET /bots
func (h *Handlers) GetBots(c echo.Context) error {
	list, err := model.GetBotsByCreator(c.Get("user").(*model.User).GetUID())
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*botForResponse, len(list))
	for i, v := range list {
		res[i] = formatBot(v)
	}

	return c.JSON(http.StatusOK, res)
}

// PostBots POST /bots
func (h *Handlers) PostBots(c echo.Context) error {
	user := c.Get("user").(*model.User)

	req := struct {
		Name            string   `json:"name"            validate:"name,max=16,required"`
		DisplayName     string   `json:"displayName"     validate:"max=32,required"`
		Description     string   `json:"description"     validate:"required"`
		PostURL         string   `json:"postUrl"         validate:"url,required"`
		SubscribeEvents []string `json:"subscribeEvents" validate:"dive,required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	iconID, err := model.GenerateIcon(req.Name)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	postURL, _ := url.Parse(req.PostURL)
	b, err := model.CreateBot(h.OAuth2, req.Name, req.DisplayName, req.Description, user.GetUID(), uuid.Must(uuid.FromString(iconID)), postURL, req.SubscribeEvents)
	if err != nil {
		switch err.(type) {
		case *validator.InvalidValidationError:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	go event.Emit(event.UserJoined, &event.UserEvent{ID: b.GetID().String()})
	return c.JSON(http.StatusCreated, formatBot(b))
}

// GetBot GET /bots/:botID
func (h *Handlers) GetBot(c echo.Context) error {
	b, err := getBot(c, uuid.FromStringOrNil(c.Param("botID")), false)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, formatBot(b))
}

// PatchBot PATCH /bots/:botID
func (h *Handlers) PatchBot(c echo.Context) error {
	req := struct {
		DisplayName string `json:"displayName"     validate:"max=64"`
		Description string `json:"description"`
		PostURL     string `json:"postUrl"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	b, err := getBot(c, uuid.FromStringOrNil(c.Param("botID")), true)
	if err != nil {
		return err
	}

	if len(req.DisplayName) > 0 {
		if err := model.UpdateBot(b.GetID(), &req.DisplayName, nil, nil, nil); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		go event.Emit(event.UserUpdated, &event.UserEvent{ID: b.GetBotUserID().String()})
	}

	if len(req.Description) > 0 {
		if err := model.UpdateBot(b.GetID(), nil, &req.Description, nil, nil); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if len(req.PostURL) > 0 {
		postURL, err := url.Parse(req.PostURL)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid postUrl")
		}

		if err := model.UpdateBot(b.GetID(), nil, nil, postURL, nil); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteBot DELETE /bots/:botID
func (h *Handlers) DeleteBot(c echo.Context) error {
	b, err := getBot(c, uuid.FromStringOrNil(c.Param("botID")), true)
	if err != nil {
		return err
	}

	if err := h.OAuth2.DeleteTokenByID(b.GetAccessTokenID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	if err := model.DeleteBot(b.GetID()); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// PutBotIcon PUT /bots/:botID/icon
func (h *Handlers) PutBotIcon(c echo.Context) error {
	b, err := getBot(c, uuid.FromStringOrNil(c.Param("botID")), true)
	if err != nil {
		return err
	}

	// file確認
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	iconID, err := processMultipartFormIconUpload(c, uploadedFile)
	if err != nil {
		return err
	}

	// アイコン変更
	if err := model.ChangeUserIcon(b.GetBotUserID(), iconID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	go event.Emit(event.UserIconUpdated, &event.UserEvent{ID: b.GetBotUserID().String()})
	return c.NoContent(http.StatusOK)
}

// PostBotActivation POST /bots/:botID/activation
func (h *Handlers) PostBotActivation(c echo.Context) error {
	b, err := getBot(c, uuid.FromStringOrNil(c.Param("botID")), true)
	if err != nil {
		return err
	}

	if err := h.Bot.ActivateBot(b.GetID()); err != nil {
		return c.JSON(http.StatusBadRequest, err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetBotToken GET /bots/:botID/token
func (h *Handlers) GetBotToken(c echo.Context) error {
	b, err := getBot(c, uuid.FromStringOrNil(c.Param("botID")), true)
	if err != nil {
		return err
	}

	at, err := h.OAuth2.GetTokenByID(b.GetAccessTokenID())
	if err != nil {
		switch err {
		case oauth2.ErrTokenNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "somehow this bot's token has been revoked. please reissue a token.")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"verificationToken": b.GetVerificationToken(),
		"accessToken":       at.AccessToken,
	})
}

// PostBotToken POST /bots/:botID/token
func (h *Handlers) PostBotToken(c echo.Context) error {
	b, err := getBot(c, uuid.FromStringOrNil(c.Param("botID")), true)
	if err != nil {
		return err
	}

	b, token, err := model.ReissueBotTokens(h.OAuth2, b.GetID())
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"verificationToken": b.GetVerificationToken(),
		"accessToken":       token,
	})
}

// GetBotInstallCode GET /bots/:botID/code
func (h *Handlers) GetBotInstallCode(c echo.Context) error {
	b, err := getBot(c, uuid.FromStringOrNil(c.Param("botID")), true)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]string{
		"installCode": b.GetInstallCode(),
	})
}

// GetInstalledBots GET /channels/:channelID/bots
func (h *Handlers) GetInstalledBots(c echo.Context) error {
	channelID := uuid.FromStringOrNil(c.Param("channelID"))
	userID := c.Get("user").(*model.User).GetUID()

	if _, err := validateChannelID(channelID, userID); err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	bots, err := model.GetInstalledBots(channelID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	res := make([]*installedBotForResponse, len(bots))
	for i, v := range bots {
		res[i] = &installedBotForResponse{
			BotID:       v.BotID,
			InstalledBy: v.InstalledBy,
			InstalledAt: v.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, res)
}

// PostInstalledBots POST /channels/:channelID/bots
func (h *Handlers) PostInstalledBots(c echo.Context) error {
	channelID := uuid.FromStringOrNil(c.Param("channelID"))
	userID := c.Get("user").(*model.User).GetUID()

	req := struct {
		Code string `json:"code" validate:"required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	b, err := model.GetBotByInstallCode(req.Code)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	if b == nil {
		return echo.NewHTTPError(http.StatusBadRequest)
	}

	if _, err := validateChannelID(channelID, userID); err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if err := model.InstallBot(b.GetID(), channelID, userID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"botId": b.GetID().String(),
	})
}

// DeleteInstalledBot DELETE /channels/:channelID/bots/:botID
func (h *Handlers) DeleteInstalledBot(c echo.Context) error {
	channelID := uuid.FromStringOrNil(c.Param("channelID"))
	user := c.Get("user").(*model.User)

	b, err := getBot(c, uuid.FromStringOrNil(c.Param("botID")), false)
	if err != nil {
		return err
	}

	if _, err := validateChannelID(channelID, user.GetUID()); err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if err := model.UninstallBot(b.GetID(), channelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

func getBot(c echo.Context, id uuid.UUID, strict bool) (model.Bot, error) {
	if id == uuid.Nil {
		return nil, echo.NewHTTPError(http.StatusNotFound)
	}

	b, err := model.GetBot(id)
	if err != nil {
		c.Logger().Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError)
	}
	if b == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound)
	}
	if strict {
		user, ok := c.Get("user").(*model.User)
		if !ok || b.GetCreatorID() != user.GetUID() {
			return nil, echo.NewHTTPError(http.StatusForbidden)
		}
	}

	return b, nil
}

func formatBot(b model.Bot) *botForResponse {
	var arr []string
	for v := range b.GetSubscribeEvents() {
		arr = append(arr, v)
	}
	return &botForResponse{
		BotID:           b.GetID().String(),
		BotUserID:       b.GetBotUserID().String(),
		Name:            b.GetName(),
		DisplayName:     b.GetDisplayName(),
		Description:     b.GetDescription(),
		SubscribeEvents: arr,
		Activated:       b.GetActivated(),
		CreatorID:       b.GetCreatorID().String(),
		CreatedAt:       b.GetCreatedAt(),
		UpdatedAt:       b.GetUpdatedAt(),
	}
}
