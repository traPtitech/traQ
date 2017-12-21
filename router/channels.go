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
		return err
	}

	channelList, err := model.GetChannels(userID)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "チャンネルリストの取得中にエラーが発生しました")
		return fmt.Errorf("Failed to get channel list: %v", err)
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
		return err
	}

	var requestBody PostChannel
	c.Bind(&requestBody)

	if requestBody.ChannelType == "" || requestBody.Name == "" {
		messageResponse(c, http.StatusBadRequest, "channelTypeまたはnameが設定されていません")
		return nil
	}

	if requestBody.ChannelType != "public" && requestBody.ChannelType != "private" {
		messageResponse(c, http.StatusBadRequest, "channelTypeはpublic privateのいずれかで設定してください")
		return nil
	}

	if requestBody.Parent != "" {
		channel := &model.Channel{ID: requestBody.Parent}
		ok, err := channel.Exists(userID)
		if err != nil {
			messageResponse(c, http.StatusInternalServerError, "親チャンネルの検証中にサーバー内でエラーが発生しました")
			return err
		}
		if !ok {
			messageResponse(c, http.StatusBadRequest, "指定された親チャンネルは存在しません")
			return nil
		}
	}

	newChannel := &model.Channel{
		CreatorID: userID,
		Name:      requestBody.Name,
		IsPublic:  requestBody.ChannelType == "public",
	}

	if err := newChannel.Create(); err != nil {
		c.Error(err)
		return err
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
				c.Error(err)
				return err
			}
		}
	}

	ch, err := model.GetChannelByID(userID, newChannel.ID)

	if err != nil {
		c.Error(err)
		return err
	}

	return c.JSON(http.StatusCreated, ch)
}

// GetChannelsByChannelID GET /channels/{channelID} のハンドラ
func GetChannelsByChannelID(c echo.Context) error {
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	channelID := c.Param("channelId")

	channel := &model.Channel{ID: channelID}
	ok, err := channel.Exists(userID)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "チャンネルの検証中にサーバー内でエラーが発生しました")
		return err
	}
	if !ok {
		messageResponse(c, http.StatusNotFound, "指定されたチャンネルは存在しません")
		return nil
	}

	childrenIDs, err := channel.Children(userID)
	if err != nil {
		c.Error(err)
		return fmt.Errorf("Failed to get children channel id list: %v", err)
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

// PutChannelsByChannelID PUT /channels/{channelId} のハンドラ
func PutChannelsByChannelID(c echo.Context) error {
	// CHECK: 権限周り
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	var requestBody PostChannel
	c.Bind(&requestBody)

	channelID := c.Param("channelId")
	channel := &model.Channel{ID: channelID}
	ok, err := channel.Exists(userID)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "チャンネルの検証中にサーバー内でエラーが発生しました")
		return err
	}
	if !ok {
		messageResponse(c, http.StatusNotFound, "指定されたチャンネルは存在しません")
		return nil
	}

	channel.Name = requestBody.Name
	channel.UpdaterID = userID

	if err := channel.Update(); err != nil {
		c.Error(err)
		return err
	}

	childrenIDs, err := channel.Children(userID)
	if err != nil {
		c.Error(err)
		return fmt.Errorf("Failed to get children channel id list: %v", err)
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

// DeleteChannelsByChannelID DELETE /channels/{channelId}のハンドラ
func DeleteChannelsByChannelID(c echo.Context) error {
	// CHECK: 権限周り
	userID, err := getUserID(c)
	if err != nil {
		return err
	}

	type confirm struct {
		Confirm bool `json:"confirm"`
	}
	var requestBody confirm
	c.Bind(&requestBody)

	channelID := c.Param("channelId")
	channel := &model.Channel{ID: channelID}
	ok, err := channel.Exists(userID)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "チャンネルの検証中にサーバー内でエラーが発生しました")
		return err
	}
	if !ok {
		messageResponse(c, http.StatusNotFound, "指定されたチャンネルは存在しません")
		return nil
	}

	if !requestBody.Confirm {
		messageResponse(c, http.StatusBadRequest, "confirmがtrueではありません")
	}
	channel.UpdaterID = userID
	channel.IsDeleted = true
	fmt.Println(channel)

	if err := channel.Update(); err != nil {
		c.Error(err)
		return err
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

func messageResponse(c echo.Context, code int, message string) {
	responseBody := ErrorResponse{}
	responseBody.Message = message
	c.JSON(code, responseBody)
}
