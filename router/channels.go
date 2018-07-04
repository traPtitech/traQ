package router

import (
	"fmt"
	"github.com/satori/go.uuid"
	"github.com/traPtitech/traQ/event"
	"net/http"

	"github.com/traPtitech/traQ/utils/validator"

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
	Member      []string `json:"member"  validate:"lte=2"`
	Name        string   `json:"name"    validate:"required"`
	Parent      string   `json:"parent"`
}

// validate 入力が正しいかどうかを検証します
func (post *PostChannel) validate(userID uuid.UUID) error {
	if err := validator.ValidateStruct(post); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	// userから親チャンネルが見えないときは追加できない
	if post.Parent != privateParentChannelID && post.Parent != "" {
		_, err := validateChannelID(uuid.FromStringOrNil(post.Parent), userID)
		if err != nil {
			switch err {
			case model.ErrNotFound:
				return echo.NewHTTPError(http.StatusBadRequest, "this parent channel is not found")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified parent channel")
			}
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

// GetChannels GET /channels のハンドラ
func GetChannels(c echo.Context) error {
	user := c.Get("user").(*model.User)

	channelList, err := model.GetChannelList(user.GetUID())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get channel list: %v", err))
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
				log.Error(err)
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get private channel members")
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

// PostChannels POST /channels のハンドラ
func PostChannels(c echo.Context) error {
	userID := c.Get("user").(*model.User).GetUID()

	req := &PostChannel{}
	if err := c.Bind(&req); err != nil {
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

// GetChannelsByChannelID GET /channels/{channelID} のハンドラ
func GetChannelsByChannelID(c echo.Context) error {
	userID := c.Get("user").(*model.User).GetUID()
	ch, err := validateChannelID(uuid.FromStringOrNil(c.Param("channelID")), userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
		}
	}

	childIDs, err := model.GetChildrenChannelIDsWithUserID(userID, ch.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get children channel id list: %v", err)
	}

	members, err := model.GetPrivateChannelMembers(ch.ID)
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get private channel members")
	}

	return c.JSON(http.StatusOK, formatChannel(ch, childIDs, members))
}

// PatchChannelsByChannelID PATCH /channels/{channelID} のハンドラ
func PatchChannelsByChannelID(c echo.Context) error {
	userID := c.Get("user").(*model.User).GetUID()
	channelID := uuid.FromStringOrNil(c.Param("channelID"))

	// チャンネル検証
	ch, err := validateChannelID(channelID, userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusBadRequest, "this channel is not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
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
	userID := c.Get("user").(*model.User).GetUID()
	channelID := uuid.FromStringOrNil(c.Param("channelID"))

	// チャンネル検証
	ch, err := validateChannelID(channelID, userID)
	if err != nil {
		switch err {
		case model.ErrNotFound:
			return echo.NewHTTPError(http.StatusBadRequest, "this channel is not found")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
		}
	}

	req := struct {
		Parent string `json:"parent"`
	}{}
	if err := bindAndValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	if _, err := uuid.FromString(req.Parent); err != nil {
		if len(req.Parent) != 0 {
			return echo.NewHTTPError(http.StatusBadRequest)
		}
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

// DeleteChannelsByChannelID DELETE /channels/{channelID}のハンドラ
func DeleteChannelsByChannelID(c echo.Context) error {
	userID := c.Get("user").(*model.User).GetUID()

	deleteQue := make([]string, 1)
	deleteQue[0] = c.Param("channelID")
	for len(deleteQue) > 0 {
		channelID := uuid.FromStringOrNil(deleteQue[0])
		deleteQue = deleteQue[1:]
		channel, err := validateChannelID(channelID, userID)
		if err != nil {
			switch err {
			case model.ErrNotFound:
				return echo.NewHTTPError(http.StatusNotFound, "this channel is not found")
			default:
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find the specified channel")
			}
		}

		children, err := model.GetChildrenChannelIDsWithUserID(userID, channel.ID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred in the server while get children channel")
		}
		deleteQue = append(deleteQue, children...)

		if err := model.DeleteChannel(channelID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred when channel model update.")
		}

		go event.Emit(event.ChannelDeleted, &event.ChannelEvent{ID: channel.ID})
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
		case model.ErrNotFound:
			return nil, nil
		}
		log.Error(err)
		return nil, err
	}

	return ch, nil
}
