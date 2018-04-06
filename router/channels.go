package router

import (
	"fmt"
	"net/http"

	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"
	"github.com/traPtitech/traQ/utils/validator"

	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"github.com/traPtitech/traQ/model"
)

const (
	// privateチャンネルが親に持つID
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
}

// PostChannel リクエストボディ用構造体
type PostChannel struct {
	ChannelType string   `json:"type"    validate:"required,oneof=public private"`
	Member      []string `json:"member"  validate:"lte=2"`
	Name        string   `json:"name"    validate:"required"`
	Parent      string   `json:"parent"`
}

// validate 入力が正しいかどうかを検証します
func (post *PostChannel) validate(userID string) error {
	if err := validator.ValidateStruct(post); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "some values are wrong")
	}

	// TODO: 同名・同階層のチャンネルのチェック

	// userから親チャンネルが見えないときは追加できない
	if post.Parent != privateParentChannelID && post.Parent != "" {
		_, err := validateChannelID(post.Parent, userID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "this parent channel is not found")
		}
	}

	if post.ChannelType == "private" {
		post.Parent = privateParentChannelID
		if len(post.Member) > 2 {
			return echo.NewHTTPError(http.StatusBadRequest, "number of private channel members should be no more than 2")
		}

		if post.Member[0] != userID && post.Member[1] != userID {
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

		pc, err := model.GetPrivateChannel(users[0], users[1])
		if err != nil && err != model.ErrNotFound {
			log.Error(err)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check the existence of the private channel")
		}
		if pc != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "the private channel exists now")
		}
	}
	return nil
}

// GetChannels GET /channels のハンドラ
func GetChannels(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	channelList, err := model.GetChannelList(userID)
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
	userID := c.Get("user").(*model.User).ID

	req := &PostChannel{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	if err := req.validate(userID); err != nil {
		return err
	}

	ch := &model.Channel{
		CreatorID: userID,
		ParentID:  req.Parent,
		Name:      req.Name,
		IsPublic:  req.ChannelType == "public",
	}
	if err := ch.Create(); err != nil {
		log.Errorf("an error occurred while create new channel: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create new channel")
	}

	if ch.IsPublic {
		go notification.Send(events.ChannelCreated, events.ChannelEvent{ID: ch.ID})
	} else {
		for _, u := range req.Member {
			upc := &model.UsersPrivateChannel{
				ChannelID: ch.ID,
				UserID:    u,
			}
			err := upc.Create()
			if err != nil {
				log.Errorf("failed to insert users_private_channel: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred while adding notified user.")
			}
		}
	}

	return c.JSON(http.StatusCreated, formatChannel(ch, []string{}, req.Member))
}

// GetChannelsByChannelID GET /channels/{channelID} のハンドラ
func GetChannelsByChannelID(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	ch, err := validateChannelID(c.Param("channelID"), userID)
	if err != nil {
		return err
	}

	childIDs, err := ch.Children(userID)
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

// PutChannelsByChannelID PUT /channels/{channelID} のハンドラ
func PutChannelsByChannelID(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	req := struct {
		Name       string `json:"name"`
		Parent     string `json:"parent"`
		Visibility bool   `json:"visibility"`
	}{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	channelID := c.Param("channelID")
	ch, err := validateChannelID(channelID, userID)
	if err != nil {
		return err
	}

	ch.Name = req.Name
	ch.ParentID = req.Parent
	ch.IsVisible = req.Visibility
	ch.UpdaterID = userID

	if err := ch.Update(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred while update channel")
	}

	childIDs, err := ch.Children(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get children channel id list: %v", err))
	}

	members, err := model.GetPrivateChannelMembers(ch.ID)
	if err != nil {
		log.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get private channel members")
	}

	go notification.Send(events.ChannelUpdated, events.ChannelEvent{ID: channelID})
	return c.JSON(http.StatusOK, formatChannel(ch, childIDs, members))
}

// DeleteChannelsByChannelID DELETE /channels/{channelID}のハンドラ
func DeleteChannelsByChannelID(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	deleteQue := make([]string, 1)
	deleteQue[0] = c.Param("channelID")
	for len(deleteQue) > 0 {
		channelID := deleteQue[0]
		deleteQue = deleteQue[1:]
		channel, err := validateChannelID(channelID, userID)
		if err != nil {
			return err
		}

		children, err := channel.Children(userID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred in the server while get children channel")
		}
		deleteQue = append(deleteQue, children...)

		channel.UpdaterID = userID
		channel.IsDeleted = true

		if err := channel.Update(); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred when channel model update.")
		}

		go notification.Send(events.ChannelDeleted, events.ChannelEvent{ID: channelID})
	}
	return c.NoContent(http.StatusNoContent)
}

func valuesChannel(m map[string]*ChannelForResponse) []*ChannelForResponse {
	arr := []*ChannelForResponse{}
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
	}
}

// リクエストされたチャンネルIDが指定されたuserから見えるかをチェックし、見える場合はそのチャンネルを返す
func validateChannelID(channelID, userID string) (*model.Channel, error) {
	ch := &model.Channel{ID: channelID}
	ok, err := ch.Exists(userID)
	if err != nil {
		log.Error(err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "An error occurred in the server while get channel")
	}
	if !ok {
		return nil, echo.NewHTTPError(http.StatusNotFound, "The specified channel does not exist")
	}
	return ch, nil
}
