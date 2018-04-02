package router

import (
	"fmt"
	"net/http"

	"github.com/traPtitech/traQ/notification"
	"github.com/traPtitech/traQ/notification/events"

	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
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

const (
	// privateチャンネルが親に持つID
	privateParentChannelID = "02472636-d850-4808-b060-8260dd6342af"
)

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

		if !ch.IsPublic {
			response[ch.ID].Parent = privateParentChannelID
		} else {
			response[ch.ID].Parent = ch.ParentID
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
	// TODO: 同名・同階層のチャンネルのチェック
	userID := c.Get("user").(*model.User).ID

	req := &PostChannel{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to bind request body.")
	}

	if req.ChannelType == "" || req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Not set channelType or name")
	}
	if req.ChannelType != "public" && req.ChannelType != "private" {
		return echo.NewHTTPError(http.StatusBadRequest, "channelType must be public or private.")
	}

	ch, err := createChannel(req.Name, userID, req.ChannelType, req.Parent, req.Member)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, formatChannel(ch))
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

	res := formatChannel(ch)
	res.Children = childIDs
	return c.JSON(http.StatusOK, res)
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

	response := ChannelForResponse{
		ChannelID:  ch.ID,
		Name:       ch.Name,
		Parent:     ch.ParentID,
		Visibility: ch.IsVisible,
		Children:   childIDs,
	}

	go notification.Send(events.ChannelUpdated, events.ChannelEvent{ID: channelID})
	return c.JSON(http.StatusOK, response)
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

func createChannel(name, creatorID, channelType, parentID string, members []string) (*model.Channel, error) {
	if parentID != "" {
		// 自分から見えないチャンネルの子チャンネルを作成することはできない
		_, err := validateChannelID(parentID, creatorID)
		if err != nil {
			return nil, err
		}
	}

	ch := &model.Channel{
		CreatorID: creatorID,
		ParentID:  parentID,
		Name:      name,
		IsPublic:  channelType == "public",
	}

	if err := ch.Create(); err != nil {
		log.Errorf("an error occurred while create new channel: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create new channel")
	}

	if ch.IsPublic {
		go notification.Send(events.ChannelCreated, events.ChannelEvent{ID: ch.ID})
	} else {
		for _, u := range members {
			upc := &model.UsersPrivateChannel{
				ChannelID: ch.ID,
				UserID:    u,
			}
			err := upc.Create()
			if err != nil {
				log.Errorf("failed to insert users_private_channel: %v", err)
				return nil, echo.NewHTTPError(http.StatusInternalServerError, "An error occurred while adding notified user.")
			}
		}
	}

	return ch, nil
}

func valuesChannel(m map[string]*ChannelForResponse) []*ChannelForResponse {
	arr := []*ChannelForResponse{}
	for _, v := range m {
		arr = append(arr, v)
	}
	return arr
}

func formatChannel(channel *model.Channel) *ChannelForResponse {
	var parentID string
	if !channel.IsPublic {
		parentID = privateParentChannelID
	} else {
		parentID = channel.ParentID
	}
	return &ChannelForResponse{
		ChannelID:  channel.ID,
		Name:       channel.Name,
		Parent:     parentID,
		Visibility: channel.IsVisible,
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
