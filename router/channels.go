package router

import (
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/traPtitech/traQ/model"
)

const (
	// DMのチャンネルが親に持つID
	privateParentChannelID = "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"
)

// ChannelForResponse レスポンス用のチャンネル構造体
type ChannelForResponse struct {
	ChannelID  string   `json:"channelId"`
	Name       string   `json:"name"`
	Parent     string   `json:"parent"`
	Children   []string `json:"children"`
	Member     []string `json:"member"`
	Visibility bool     `json:"visibility"`
	Force      bool     `json:"force"`
}

// PostChannel リクエストボディ用構造体
type PostChannel struct {
	// TODO: DM以外のprivateチャンネルに対応する場合は修正が必要
	ChannelType string   `json:"type"    validate:"required,oneof=public private"`
	Member      []string `json:"member"  validate:"lte=2,dive,required"`
	Name        string   `json:"name"    validate:"required"`
	Parent      string   `json:"parent"`
}

// validate 入力が正しいかどうかを検証します
func (post *PostChannel) validate(userID uuid.UUID) error {
	// userから親チャンネルが見えないときは追加できない
	if post.Parent != privateParentChannelID && post.Parent != "" {
		if ok, err := model.IsChannelAccessibleToUser(userID, uuid.FromStringOrNil(post.Parent)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError)
		} else if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "this parent channel is not found")
		}
	}

	if post.ChannelType == "private" {
		// TODO: DM以外のprivateチャンネルに対応する場合は修正が必要
		post.Parent = privateParentChannelID
		if len(post.Member) > 2 {
			return echo.NewHTTPError(http.StatusBadRequest, "number of private channel members should be no more than 2")
		}

		if post.Member[0] != userID.String() && post.Member[1] != userID.String() {
			return echo.NewHTTPError(http.StatusBadRequest, "you should join this private channel")
		}

		// DMが既に存在する場合はエラー
		var users [2]string
		switch len(post.Member) {
		case 1:
			users[0] = post.Member[0]
			users[1] = post.Member[0]
		case 2:
			users[0] = post.Member[0]
			users[1] = post.Member[1]
		default:
			return echo.NewHTTPError(http.StatusBadRequest, "number of private channel members should be no more than 2")
		}

		pcID, err := model.GetPrivateChannel(users[0], users[1])
		if err != nil && err != model.ErrNotFound {
			log.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check the existence of the private channel")
		}
		if pcID != "" {
			return echo.NewHTTPError(http.StatusBadRequest, "the private channel exists now")
		}
	}

	return nil
}

// GetChannels GET /channels
func GetChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	channelList, err := model.GetChannelList(userID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	response := make(map[string]*ChannelForResponse)
	for _, ch := range channelList {
		if response[ch.ID] == nil {
			response[ch.ID] = &ChannelForResponse{}
		}
		response[ch.ID].ChannelID = ch.ID
		response[ch.ID].Name = ch.Name
		response[ch.ID].Visibility = ch.IsVisible
		response[ch.ID].Parent = ch.ParentID
		response[ch.ID].Force = ch.IsForced

		if !ch.IsPublic {
			member, err := model.GetPrivateChannelMembers(ch.ID)
			if err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
			response[ch.ID].Member = member
		} else {
			if response[ch.ParentID] == nil {
				response[ch.ParentID] = &ChannelForResponse{}
			}
			response[ch.ParentID].Children = append(response[ch.ParentID].Children, ch.ID)
		}
	}

	return c.JSON(http.StatusOK, valuesChannel(response))
}

// PostChannels POST /channels
func PostChannels(c echo.Context) error {
	userID := getRequestUserID(c)

	req := &PostChannel{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := req.validate(userID); err != nil {
		return err
	}

	ch, err := model.CreateChannel(req.Parent, req.Name, userID, req.ChannelType == "public")
	if err != nil {
		switch err {
		case model.ErrDuplicateName:
			return echo.NewHTTPError(http.StatusConflict, err)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if ch.IsPublic {
		go event.Emit(event.ChannelCreated, &event.ChannelEvent{ID: ch.ID})
	} else {
		for _, u := range req.Member {
			user, err := model.GetUser(uuid.FromStringOrNil(u))
			if err != nil {
				switch err {
				case model.ErrNotFound:
					continue
				default:
					c.Logger().Error(err)
					return echo.NewHTTPError(http.StatusInternalServerError)
				}
			}

			if err := model.AddPrivateChannelMember(ch.GetCID(), user.GetUID()); err != nil {
				c.Logger().Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}
		}

		ids := make([]uuid.UUID, len(req.Member))
		for k, v := range req.Member {
			ids[k] = uuid.Must(uuid.FromString(v))
		}
		go event.Emit(event.ChannelCreated, &event.PrivateChannelEvent{UserIDs: ids, ChannelID: ch.GetCID()})
	}

	return c.JSON(http.StatusCreated, formatChannel(ch, []string{}, req.Member))
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

	childIDs, err := model.GetChildrenChannelIDsWithUserID(userID, ch.ID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	members, err := model.GetPrivateChannelMembers(ch.ID)
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusOK, formatChannel(ch, childIDs, members))
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
			return echo.NewHTTPError(http.StatusBadRequest, "this channel is not found")
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
		users, err := model.GetPrivateChannelMembers(ch.ID)
		if err != nil {
			c.Logger().Error(err)
		}
		ids := make([]uuid.UUID, len(users))
		for i, v := range users {
			ids[i] = uuid.Must(uuid.FromString(v))
		}
		go event.Emit(event.ChannelUpdated, &event.PrivateChannelEvent{UserIDs: ids, ChannelID: channelID})
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
			return echo.NewHTTPError(http.StatusBadRequest, "this channel is not found")
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	req := struct {
		Parent string `json:"parent" validate:"required,uuid"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if err := model.ChangeChannelParent(channelID, req.Parent, userID); err != nil {
		switch err {
		case model.ErrDuplicateName:
			return echo.NewHTTPError(http.StatusConflict, err)
		case model.ErrChannelPathDepth:
			return echo.NewHTTPError(http.StatusBadRequest, err)
		default:
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
	}

	if ch.IsPublic {
		go event.Emit(event.ChannelUpdated, &event.ChannelEvent{ID: ch.ID})
	} else {
		users, err := model.GetPrivateChannelMembers(ch.ID)
		if err != nil {
			c.Logger().Error(err)
		}
		ids := make([]uuid.UUID, len(users))
		for i, v := range users {
			ids[i] = uuid.Must(uuid.FromString(v))
		}
		go event.Emit(event.ChannelUpdated, &event.PrivateChannelEvent{UserIDs: ids, ChannelID: channelID})
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

	deleteQue := make([]string, 1)
	deleteQue[0] = channelID.String()
	for len(deleteQue) > 0 {
		channelID := uuid.FromStringOrNil(deleteQue[0])
		deleteQue = deleteQue[1:]

		children, err := model.GetChildrenChannelIDs(channelID)
		if err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}
		deleteQue = append(deleteQue, children...)

		if err := model.DeleteChannel(channelID); err != nil {
			c.Logger().Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError)
		}

		go event.Emit(event.ChannelDeleted, &event.ChannelEvent{ID: channelID.String()})
	}
	return c.NoContent(http.StatusNoContent)
}

func valuesChannel(m map[string]*ChannelForResponse) (arr []*ChannelForResponse) {
	for _, v := range m {
		arr = append(arr, v)
	}
	return arr
}

func formatChannel(channel *model.Channel, childIDs, members []string) *ChannelForResponse {
	return &ChannelForResponse{
		ChannelID:  channel.ID,
		Name:       channel.Name,
		Visibility: channel.IsVisible,
		Parent:     channel.ParentID,
		Member:     members,
		Children:   childIDs,
		Force:      channel.IsForced,
	}
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
