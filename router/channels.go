package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/traPtitech/traQ/model"
)

// ChannelForResponse レスポンス用のチャンネル構造体
type ChannelForResponse struct {
	ChannelID  string
	Name       string
	Parent     string
	Children   []string
	Visibility bool
}

// ErrorResponse エラーレスポンス用の構造体
type ErrorResponse struct {
	Message string
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
	userID, err := getUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("An error occuerred while getUserID: %v", err))
	}

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
		response[ch.ID].Visibility = !ch.IsVisible

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
	userID, err := getUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("An error occuerred while getUserID: %v", err))
	}

	var requestBody PostChannel
	c.Bind(&requestBody)

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
		Name:      requestBody.Name,
		IsPublic:  requestBody.ChannelType == "public",
	}

	if err := newChannel.Create(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("An error occurred while create new channel: %v", err))
	}

	if newChannel.IsPublic {
		// TODO:通知周りの実装
	} else {
		for _, user := range requestBody.Member {
			// TODO: メンバーが存在するか確認
			usersPrivateChannel := &model.UsersPrivateChannel{}
			usersPrivateChannel.ChannelID = newChannel.ID
			usersPrivateChannel.UserID = user
			err := usersPrivateChannel.Create()
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred while adding notificated user.")
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
	userID, err := getUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("An error occuerred while getUserID: %v", err))
	}

	channelID := c.Param("channelId")

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

	response := ChannelForResponse{
		ChannelID:  channel.ID,
		Name:       channel.Name,
		Parent:     channel.ParentID,
		Visibility: !channel.IsVisible,
		Children:   childrenIDs,
	}

	return c.JSON(http.StatusOK, response)
}

// PutChannelsByChannelID PUT /channels/{channelID} のハンドラ
func PutChannelsByChannelID(c echo.Context) error {
	// CHECK: 権限周り
	userID, err := getUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("An error occuerred while getUserID: %v", err))
	}

	var requestBody PostChannel
	c.Bind(&requestBody)

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
		return echo.NewHTTPError(http.StatusInternalServerError, "An error occuerred while update channel")
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

	return c.JSON(http.StatusOK, response)
}

// DeleteChannelsByChannelID DELETE /channels/{channelID}のハンドラ
func DeleteChannelsByChannelID(c echo.Context) error {
	// CHECK: 権限周り
	userID, err := getUserID(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("An error occuerred while getUserID: %v", err))
	}

	type confirm struct {
		Confirm bool `json:"confirm"`
	}
	var requestBody confirm
	c.Bind(&requestBody)

	channelID := c.Param("channelID")
	channel := &model.Channel{ID: channelID}
	ok, err := channel.Exists(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "An error occurred in the server while verifying the channel.")
	}
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound, "The specified channel does not exist.")
	}

	if !requestBody.Confirm {
		return echo.NewHTTPError(http.StatusBadRequest, "confirm is not true.")
	}
	channel.UpdaterID = userID
	channel.IsDeleted = true
	fmt.Println(channel)

	if err := channel.Update(); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "An error occuerred when channel model update.")
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
