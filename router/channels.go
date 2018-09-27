package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
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
	Parent  string      `json:"parent"`
	Private bool        `json:"private"`
	Members []uuid.UUID `json:"member"`
}

// GetChannels GET /channels
func GetChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	channelList, err := model.GetChannelList(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	chMap := make(map[string]*ChannelForResponse, len(channelList))
	for _, ch := range channelList {
		entry, ok := chMap[ch.ID]
		if !ok {
			entry = &ChannelForResponse{}
			chMap[ch.ID] = entry
		}

		entry.ChannelID = ch.ID
		entry.Name = ch.Name
		entry.Visibility = ch.IsVisible
		entry.Parent = ch.ParentID
		entry.Force = ch.IsForced
		entry.Private = !ch.IsPublic
		entry.DM = ch.IsDMChannel()

		if !ch.IsPublic {
			// プライベートチャンネルのメンバー取得
			member, err := model.GetPrivateChannelMembers(ch.GetCID())
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			entry.Member = member
		}

		parent, ok := chMap[ch.ParentID]
		if !ok {
			parent = &ChannelForResponse{}
			chMap[ch.ParentID] = parent
		}
		parent.Children = append(parent.Children, ch.GetCID())
	}

	res := make([]*ChannelForResponse, 0, len(chMap))
	for _, v := range chMap {
		res = append(res, v)
	}
	return c.JSON(http.StatusOK, res)
}

// PostChannels POST /channels
func PostChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	req := PostChannel{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// 親チャンネルがユーザーから見えないと作成できない
	if len(req.Parent) > 0 {
		if ok, err := model.IsChannelAccessibleToUser(userID, uuid.FromStringOrNil(req.Parent)); err != nil {
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
		ch, err = model.CreatePrivateChannel(req.Parent, req.Name, userID, req.Members)
		if err != nil {
			switch err {
			case model.ErrDuplicateName:
				return echo.NewHTTPError(http.StatusConflict, err)
			case model.ErrParentChannelDifferentOpenStatus:
				return echo.NewHTTPError(http.StatusForbidden)
			case model.ErrChannelDepthLimitation:
				return echo.NewHTTPError(http.StatusBadRequest, err)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
		go event.Emit(event.ChannelCreated, &event.ChannelEvent{ID: ch.ID})
	} else {
		// 公開チャンネル
		ch, err = model.CreatePublicChannel(req.Parent, req.Name, userID)
		if err != nil {
			switch err {
			case model.ErrDuplicateName:
				return echo.NewHTTPError(http.StatusConflict, err)
			case model.ErrParentChannelDifferentOpenStatus:
				return echo.NewHTTPError(http.StatusForbidden)
			case model.ErrChannelDepthLimitation:
				return echo.NewHTTPError(http.StatusBadRequest, err)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
		go event.Emit(event.ChannelCreated, &event.PrivateChannelEvent{ChannelID: ch.GetCID()})
	}

	formatted, err := formatChannel(ch)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, formatted)
}

// GetChannelByChannelID GET /channels/:channelID
func GetChannelByChannelID(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	ch, err := validateChannelID(channelID, userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	formatted, err := formatChannel(ch)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, formatted)
}

// PatchChannelByChannelID PATCH /channels/:channelID
func PatchChannelByChannelID(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	// チャンネル検証
	ch, err := validateChannelID(channelID, userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	req := struct {
		Name       *string `json:"name"`
		Visibility *bool   `json:"visibility"`
		Force      *bool   `json:"force"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if req.Name != nil && len(*req.Name) > 0 {
		if err := model.ChangeChannelName(channelID, *req.Name, userID); err != nil {
			switch err {
			case model.ErrDuplicateName:
				return echo.NewHTTPError(http.StatusConflict, err)
			case model.ErrForbidden:
				return echo.NewHTTPError(http.StatusForbidden)
			default:
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}
	}

	if req.Force != nil || req.Visibility != nil {
		if err := model.UpdateChannelFlag(channelID, req.Visibility, req.Force, userID); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if ch.IsPublic {
		go event.Emit(event.ChannelUpdated, &event.ChannelEvent{ID: ch.ID})
	} else {
		go event.Emit(event.ChannelUpdated, &event.PrivateChannelEvent{ChannelID: channelID})
	}
	return c.NoContent(http.StatusNoContent)
}

// PutChannelParent PUT /channels/:channelID/parent
func PutChannelParent(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	// チャンネル検証
	ch, err := validateChannelID(channelID, userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	req := struct {
		Parent string `json:"parent"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := model.ChangeChannelParent(channelID, req.Parent, userID); err != nil {
		switch err {
		case model.ErrDuplicateName:
			return echo.NewHTTPError(http.StatusConflict, err)
		case model.ErrChannelDepthLimitation:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		case model.ErrForbidden:
			return echo.NewHTTPError(http.StatusForbidden)
		case model.ErrDirectMessageChannelCannotHaveChildren:
			return echo.NewHTTPError(http.StatusForbidden)
		case model.ErrParentChannelDifferentOpenStatus:
			return echo.NewHTTPError(http.StatusForbidden)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if ch.IsPublic {
		go event.Emit(event.ChannelUpdated, &event.ChannelEvent{ID: ch.ID})
	} else {
		go event.Emit(event.ChannelUpdated, &event.PrivateChannelEvent{ChannelID: channelID})
	}

	return c.NoContent(http.StatusNoContent)
}

// DeleteChannelByChannelID DELETE /channels/:channelID
func DeleteChannelByChannelID(c echo.Context) error {
	userID := getRequestUserID(c)
	channelID := getRequestParamAsUUID(c, paramChannelID)

	if ok, err := model.IsChannelAccessibleToUser(userID, channelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	if err := model.DeleteChannel(channelID); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

// リクエストされたチャンネルIDが指定されたuserから見えるかをチェックし、見える場合はそのチャンネルを返す
func validateChannelID(channelID, userID uuid.UUID) (*model.Channel, error) {
	ch, err := model.GetChannelWithUserID(userID, channelID)
	if err != nil {
		switch err {
		case model.ErrNotFoundOrForbidden:
			return nil, model.ErrNotFound
		}
		return nil, err
	}

	return ch, nil
}

func formatChannel(channel *model.Channel) (response *ChannelForResponse, err error) {
	response = &ChannelForResponse{
		ChannelID:  channel.ID,
		Name:       channel.Name,
		Visibility: channel.IsVisible,
		Parent:     channel.ParentID,
		Force:      channel.IsForced,
		Private:    !channel.IsPublic,
		DM:         channel.IsDMChannel(),
	}
	response.Children, err = model.GetChildrenChannelIDs(channel.GetCID())
	if err != nil {
		return nil, err
	}

	if response.Private {
		response.Member, err = model.GetPrivateChannelMembers(channel.GetCID())
		if err != nil {
			return nil, err
		}
	}

	return response, nil
}
