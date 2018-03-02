package router

import (
	"fmt"
	"net/http"

	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// ChannelForResponse レスポンス用のチャンネル構造体
type ChannelForResponse struct {
	ChannelID  string   `json:"channelId"`
	Name       string   `json:"name"`
	Parent     string   `json:"parent"`
	Children   []string `json:"children"`
	Visibility bool     `json:"visibility"`
}

// PostChannel リクエストボディ用構造体
type PostChannel struct {
	ChannelType string   `json:"type"`
	Member      []string `json:"member"`
	Name        string   `json:"name"`
	Parent      string   `json:"parent"`
}

// GetChannels GET /channels のハンドラ
func GetChannels(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID

	channelList, err := model.GetChannels(userID)
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
		response[ch.ID].Parent = ch.ParentID
		response[ch.ID].Visibility = ch.IsVisible

		if response[ch.ParentID] == nil {
			response[ch.ParentID] = &ChannelForResponse{}
		}
		response[ch.ParentID].Children = append(response[ch.ParentID].Children, ch.ID)
	}

	return c.JSON(http.StatusOK, valuesChannel(response))
}

// PostChannels POST /channels のハンドラ
func PostChannels(c echo.Context) error {
	// TODO: 同名・同階層のチャンネルのチェック
	userID := c.Get("user").(*model.User).ID

	var requestBody PostChannel
	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	if requestBody.ChannelType == "" || requestBody.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Not set channelType or name")
	}

	if requestBody.ChannelType != "public" && requestBody.ChannelType != "private" {
		return echo.NewHTTPError(http.StatusBadRequest, "channelType must be public or private.")
	}

	if requestBody.Parent != "" {
		channel := &model.Channel{ID: requestBody.Parent}
		ok, err := channel.Exists(userID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred in the server during validation of the parent channel.")
		}
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "Not found parent channel.")
		}
	}

	newChannel := &model.Channel{
		CreatorID: userID,
		ParentID:  requestBody.Parent,
		Name:      requestBody.Name,
		IsPublic:  requestBody.ChannelType == "public",
	}

	if err := newChannel.Create(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("An error occurred while create new channel: %v", err))
	}

	if newChannel.IsPublic {
		go notification.Send(events.ChannelCreated, events.ChannelEvent{ID: newChannel.ID})
	} else {
		for _, user := range requestBody.Member {
			// TODO: メンバーが存在するか確認
			usersPrivateChannel := &model.UsersPrivateChannel{}
			usersPrivateChannel.ChannelID = newChannel.ID
			usersPrivateChannel.UserID = user
			err := usersPrivateChannel.Create()
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred while adding notified user.")
			}
		}
	}

	ch, err := model.GetChannelByID(userID, newChannel.ID)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred while getting channel.")
	}

	return c.JSON(http.StatusCreated, ch)
}

// GetChannelsByChannelID GET /channels/{channelID} のハンドラ
func GetChannelsByChannelID(c echo.Context) error {
	userID := c.Get("user").(*model.User).ID
	channelID := c.Param("channelID")

	channel := &model.Channel{ID: channelID}
	ok, err := channel.Exists(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred in the server while verifying the channel.")
	}
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "The specified channel does not exist.")
	}

	childrenIDs, err := channel.Children(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get children channel id list: %v", err)
	}

	responseBody := formatChannel(channel)
	responseBody.Children = childrenIDs
	return c.JSON(http.StatusOK, responseBody)
}

// PutChannelsByChannelID PUT /channels/{channelID} のハンドラ
func PutChannelsByChannelID(c echo.Context) error {
	// CHECK: 権限周り
	userID := c.Get("user").(*model.User).ID

	var requestBody PostChannel
	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	channelID := c.Param("channelID")
	channel := &model.Channel{ID: channelID}
	ok, err := channel.Exists(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred in the server while verifying the channel.")
	}
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "The specified channel does not exist.")
	}

	channel.Name = requestBody.Name
	channel.UpdaterID = userID

	if err := channel.Update(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred while update channel")
	}

	childrenIDs, err := channel.Children(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get children channel id list: %v", err))
	}

	response := ChannelForResponse{
		ChannelID:  channel.ID,
		Name:       channel.Name,
		Parent:     channel.ParentID,
		Visibility: !channel.IsVisible,
		Children:   childrenIDs,
	}

	go notification.Send(events.ChannelUpdated, events.ChannelEvent{ID: channelID})
	return c.JSON(http.StatusOK, response)
}

// DeleteChannelsByChannelID DELETE /channels/{channelID}のハンドラ
func DeleteChannelsByChannelID(c echo.Context) error {
	// CHECK: 権限周り
	userID := c.Get("user").(*model.User).ID

	type confirm struct {
		Confirm bool `json:"confirm"`
	}
	var requestBody confirm
	if err := c.Bind(&requestBody); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	if !requestBody.Confirm {
		return echo.NewHTTPError(http.StatusBadRequest, "confirm is not true.")
	}

	deleteQue := make([]string, 1)
	deleteQue[0] = c.Param("channelID")
	for len(deleteQue) > 0 {
		channelID := deleteQue[0]
		deleteQue = deleteQue[1:]
		fmt.Println(len(deleteQue))
		channel := &model.Channel{ID: channelID}
		ok, err := channel.Exists(userID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred in the server while verifying the channel.")
		}
		if !ok {
			return echo.NewHTTPError(http.StatusNotFound, "The specified channel does not exist.")
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

func formatChannel(channel *model.Channel) *ChannelForResponse {
	return &ChannelForResponse{
		ChannelID:  channel.ID,
		Name:       channel.Name,
		Parent:     channel.ParentID,
		Visibility: channel.IsVisible,
	}
}
