package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/repository"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// ChannelForResponse レスポンス用のチャンネル構造体
type ChannelForResponse struct {
	ChannelID  string      `json:"channelId"`
	Name       string      `json:"name"`
	Parent     string      `json:"parent"`
	Children   []uuid.UUID `json:"children"`
	Member     []uuid.UUID `json:"member"`
	Visibility bool        `json:"visibility"`
	Force      bool        `json:"force"`
	Private    bool        `json:"private"`
	DM         bool        `json:"dm"`
}

// PostChannel リクエストボディ用構造体
type PostChannel struct {
	Name    string      `json:"name"    validate:"channel,required"`
	Parent  uuid.UUID   `json:"parent"`
	Private bool        `json:"private"`
	Members []uuid.UUID `json:"member"`
}

// GetChannels GET /channels
func (h *Handlers) GetChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	channelList, err := h.Repo.GetChannelsByUserID(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	chMap := make(map[string]*ChannelForResponse, len(channelList))
	for _, ch := range channelList {
		entry, ok := chMap[ch.ID.String()]
		if !ok {
			entry = &ChannelForResponse{}
			chMap[ch.ID.String()] = entry
		}

		entry.ChannelID = ch.ID.String()
		entry.Name = ch.Name
		entry.Visibility = ch.IsVisible
		entry.Force = ch.IsForced
		entry.Private = !ch.IsPublic
		entry.DM = ch.IsDMChannel()

		if !ch.IsPublic {
			// プライベートチャンネルのメンバー取得
			member, err := h.Repo.GetPrivateChannelMemberIDs(ch.ID)
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			entry.Member = member
		}

		if ch.ParentID != uuid.Nil {
			entry.Parent = ch.ParentID.String()
			parent, ok := chMap[ch.ParentID.String()]
			if !ok {
				parent = &ChannelForResponse{}
				chMap[ch.ParentID.String()] = parent
			}
			parent.Children = append(parent.Children, ch.ID)
		} else {
			parent, ok := chMap[""]
			if !ok {
				parent = &ChannelForResponse{}
				chMap[""] = parent
			}
			parent.Children = append(parent.Children, ch.ID)
		}
	}

	res := make([]*ChannelForResponse, 0, len(chMap))
	for _, v := range chMap {
		res = append(res, v)
	}
	return c.JSON(http.StatusOK, res)
}

// PostChannels POST /channels
func (h *Handlers) PostChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	req := PostChannel{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// 親チャンネルがユーザーから見えないと作成できない
	if req.Parent != uuid.Nil {
		if ok, err := h.Repo.IsChannelAccessibleToUser(userID, req.Parent); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		} else if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "this parent channel is not found")
		}
	}

	var (
		ch  *model.Channel
		err error
	)

	if req.Private {
		// 非公開チャンネル
		ch, err = h.Repo.CreatePrivateChannel(req.Name, userID, req.Members)
		if err != nil {
			switch err {
			case repository.ErrAlreadyExists:
				return echo.NewHTTPError(http.StatusConflict, err)
			case repository.ErrForbidden:
				return echo.NewHTTPError(http.StatusForbidden)
			case repository.ErrChannelDepthLimitation:
				return echo.NewHTTPError(http.StatusBadRequest, err)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
	} else {
		// 公開チャンネル
		ch, err = h.Repo.CreatePublicChannel(req.Name, req.Parent, userID)
		if err != nil {
			switch err {
			case repository.ErrAlreadyExists:
				return echo.NewHTTPError(http.StatusConflict, err)
			case repository.ErrForbidden:
				return echo.NewHTTPError(http.StatusForbidden)
			case repository.ErrChannelDepthLimitation:
				return echo.NewHTTPError(http.StatusBadRequest, err)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
	}

	formatted, err := h.formatChannel(ch)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, formatted)
}

// GetChannelByChannelID GET /channels/:channelID
func (h *Handlers) GetChannelByChannelID(c echo.Context) error {
	ch := getChannelFromContext(c)

	formatted, err := h.formatChannel(ch)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusOK, formatted)
}

// PatchChannelByChannelID PATCH /channels/:channelID
func (h *Handlers) PatchChannelByChannelID(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, paramChannelID)

	req := struct {
		Name       *string `json:"name"`
		Visibility *bool   `json:"visibility"`
		Force      *bool   `json:"force"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if req.Name != nil && len(*req.Name) > 0 {
		if err := h.Repo.ChangeChannelName(channelID, *req.Name); err != nil {
			switch err {
			case repository.ErrAlreadyExists:
				return echo.NewHTTPError(http.StatusConflict, err)
			case repository.ErrForbidden:
				return echo.NewHTTPError(http.StatusForbidden)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
	}

	if req.Force != nil || req.Visibility != nil {
		if err := h.Repo.UpdateChannelAttributes(channelID, req.Visibility, req.Force); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}
	return c.NoContent(http.StatusNoContent)
}

// PostChannelChildren POST /channels/:channelID/children
func (h *Handlers) PostChannelChildren(c echo.Context) error {
	userID := getRequestUserID(c)
	parentCh := getChannelFromContext(c)

	var req struct {
		Name string `json:"name" validate:"channel,required"`
	}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// 子チャンネル作成
	ch, err := h.Repo.CreateChildChannel(req.Name, parentCh.ID, userID)
	if err != nil {
		switch err {
		case repository.ErrAlreadyExists:
			return echo.NewHTTPError(http.StatusConflict, err)
		case repository.ErrForbidden:
			return echo.NewHTTPError(http.StatusForbidden)
		case repository.ErrChannelDepthLimitation:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	formatted, err := h.formatChannel(ch)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	return c.JSON(http.StatusCreated, formatted)
}

// PutChannelParent PUT /channels/:channelID/parent
func (h *Handlers) PutChannelParent(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, paramChannelID)

	req := struct {
		Parent string `json:"parent" validate:"uuid,required"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := h.Repo.ChangeChannelParent(channelID, uuid.FromStringOrNil(req.Parent)); err != nil {
		switch err {
		case repository.ErrAlreadyExists:
			return echo.NewHTTPError(http.StatusConflict, err)
		case repository.ErrChannelDepthLimitation:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		case repository.ErrForbidden:
			return echo.NewHTTPError(http.StatusForbidden)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteChannelByChannelID DELETE /channels/:channelID
func (h *Handlers) DeleteChannelByChannelID(c echo.Context) error {
	channelID := getRequestParamAsUUID(c, paramChannelID)

	if err := h.Repo.DeleteChannel(channelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetTopic GET /channels/:channelID/topic
func (h *Handlers) GetTopic(c echo.Context) error {
	ch := getChannelFromContext(c)
	return c.JSON(http.StatusOK, map[string]string{
		"text": ch.Topic,
	})
}

// PutTopic PUT /channels/:channelID/topic
func (h *Handlers) PutTopic(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	req := struct {
		Text string `json:"text"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := h.Repo.UpdateChannelTopic(channelID, req.Text, userID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handlers) formatChannel(channel *model.Channel) (response *ChannelForResponse, err error) {
	response = &ChannelForResponse{
		ChannelID:  channel.ID.String(),
		Name:       channel.Name,
		Visibility: channel.IsVisible,
		Force:      channel.IsForced,
		Private:    !channel.IsPublic,
		DM:         channel.IsDMChannel(),
		Member:     make([]uuid.UUID, 0),
	}
	if channel.ParentID != uuid.Nil {
		response.Parent = channel.ParentID.String()
	}
	response.Children, err = h.Repo.GetChildrenChannelIDs(channel.ID)
	if err != nil {
		return nil, err
	}

	if response.Private {
		response.Member, err = h.Repo.GetPrivateChannelMemberIDs(channel.ID)
		if err != nil {
			return nil, err
		}
	}

	return response, nil
}
