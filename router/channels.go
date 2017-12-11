package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
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

// GetChannelsHandler GET /channels のハンドラ
func GetChannelsHandler(c echo.Context) error {
	sess, err := session.Get("sessions", c)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "セッションの取得に失敗しました")
		return fmt.Errorf("Failed to get session: %v", err)
	}

	var userID string
	if sess.Values["userId"] != nil {
		userID = sess.Values["userId"].(string)
	}
	channelList, err := model.GetChannelList(userID)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "チャンネルリストの取得中にエラーが発生しました")
		return fmt.Errorf("Failed to get channel list: %v", err)
	}

	response := make(map[string]*ChannelForResponse)

	for _, ch := range channelList {
		if response[ch.ID] == nil {
			response[ch.ID] = new(ChannelForResponse)
		}
		response[ch.ID].ChannelID = ch.ID
		response[ch.ID].Name = ch.Name
		response[ch.ID].Parent = ch.ParentID
		response[ch.ID].Visibility = !ch.IsHidden

		if response[ch.ParentID] == nil {
			response[ch.ParentID] = new(ChannelForResponse)
		}
		response[ch.ParentID].Children = append(response[ch.ParentID].Children, ch.ID)
	}

	c.JSON(http.StatusOK, values(response))
	return nil
}

// PostChannelsHandler POST /channels のハンドラ
func PostChannelsHandler(c echo.Context) error {
	// TODO: 同名・同階層のチャンネルのチェック
	sess, err := session.Get("sessions", c)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "セッションの取得に失敗しました")
		return fmt.Errorf("Failed to get session: %v", err)
	}

	var userID string
	if sess.Values["userId"] != nil {
		userID = sess.Values["userId"].(string)
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
		ok, err := model.ExistsChannel(requestBody.Parent)
		if err != nil {
			messageResponse(c, http.StatusInternalServerError, "親チャンネルの検証中にサーバー内でエラーが発生しました")
			return err
		}
		if !ok {
			messageResponse(c, http.StatusBadRequest, "指定された親チャンネルは存在しません")
			return nil
		}
	}

	newChannel := new(model.Channel)
	newChannel.CreatorID = userID
	newChannel.Name = requestBody.Name

	if requestBody.ChannelType == "public" {
		newChannel.IsPublic = true
	} else {
		newChannel.IsPublic = false
	}

	err = newChannel.Create()
	if err != nil {
		c.Error(err)
		return err
	}

	if requestBody.ChannelType == "public" {
		// TODO:通知周りの実装
	} else {
		for _, user := range requestBody.Member {
			// TODO: メンバーが存在するか確認
			usersPrivateChannel := new(model.UsersPrivateChannel)
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

	c.JSON(http.StatusCreated, ch)
	return nil
}

// GetChannelsByChannelIDHandler GET /channels/{channelID} のハンドラ
func GetChannelsByChannelIDHandler(c echo.Context) error {
	sess, err := session.Get("sessions", c)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "セッションの取得に失敗しました")
		return fmt.Errorf("Failed to get session: %v", err)
	}

	var userID string
	if sess.Values["userId"] != nil {
		userID = sess.Values["userId"].(string)
	}

	channelID := c.Param("channelId")

	ok, err := model.ExistsChannel(channelID)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "チャンネルの検証中にサーバー内でエラーが発生しました")
		return err
	}
	if !ok {
		messageResponse(c, http.StatusNotFound, "指定されたチャンネルは存在しません")
		return nil
	}

	channel, err := model.GetChannelByID(userID, channelID)
	if err != nil {
		c.Error(err)
		return fmt.Errorf("Failed to get channel: %v", err)
	}

	childrenIDList, err := model.GetChildrenChannelIDList(userID, channel.ID)
	if err != nil {
		c.Error(err)
		return fmt.Errorf("Failed to get children channel id list: %v", err)
	}

	response := ChannelForResponse{
		ChannelID:  channel.ID,
		Name:       channel.Name,
		Parent:     channel.ParentID,
		Visibility: !channel.IsHidden,
		Children:   childrenIDList,
	}

	c.JSON(http.StatusOK, response)
	return nil
}

// PutChannelsByChannelIDHandler PUT /channels/{channelId} のハンドラ
func PutChannelsByChannelIDHandler(c echo.Context) error {
	// CHECK: 権限周り
	// TODO: 必要な引数があるかチェック
	sess, err := session.Get("sessions", c)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "セッションの取得に失敗しました")
		return fmt.Errorf("Failed to get session: %v", err)
	}

	var userID string
	if sess.Values["userId"] != nil {
		userID = sess.Values["userId"].(string)
	}
	var requestBody PostChannel
	c.Bind(&requestBody)

	channelID := c.Param("channelId")
	ok, err := model.ExistsChannel(channelID)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "チャンネルの検証中にサーバー内でエラーが発生しました")
		return err
	}
	if !ok {
		messageResponse(c, http.StatusNotFound, "指定されたチャンネルは存在しません")
		return nil
	}

	channel, err := model.GetChannelByID(userID, channelID)
	if err != nil {
		c.Error(err)
		return fmt.Errorf("Failed to get channel: %v", err)
	}

	channel.Name = requestBody.Name
	channel.UpdaterID = userID

	if err := channel.Update(); err != nil {
		c.Error(err)
		return err
	}

	childrenIDList, err := model.GetChildrenChannelIDList(userID, channel.ID)
	if err != nil {
		c.Error(err)
		return fmt.Errorf("Failed to get children channel id list: %v", err)
	}

	response := ChannelForResponse{
		ChannelID:  channel.ID,
		Name:       channel.Name,
		Parent:     channel.ParentID,
		Visibility: !channel.IsHidden,
		Children:   childrenIDList,
	}

	c.JSON(http.StatusOK, response)
	return nil
}

// DeleteChannelsByChannelIDHandler DELETE /channels/{channelId}のハンドラ
func DeleteChannelsByChannelIDHandler(c echo.Context) error {
	// CHECK: 権限周り
	sess, err := session.Get("sessions", c)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "セッションの取得に失敗しました")
		return fmt.Errorf("Failed to get session: %v", err)
	}

	var userID string
	if sess.Values["userId"] != nil {
		userID = sess.Values["userId"].(string)
	}
	type confirm struct {
		Confirm bool `json:"confirm"`
	}
	var requestBody confirm
	c.Bind(&requestBody)

	channelID := c.Param("channelId")
	ok, err := model.ExistsChannel(channelID)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "チャンネルの検証中にサーバー内でエラーが発生しました")
		return err
	}
	if !ok {
		messageResponse(c, http.StatusNotFound, "指定されたチャンネルは存在しません")
		return nil
	}

	channel, err := model.GetChannelByID(userID, channelID)
	if err != nil {
		c.Error(err)
		return fmt.Errorf("Failed to get channel: %v", err)
	}

	if requestBody.Confirm {
		channel.UpdaterID = userID
		channel.IsDeleted = true
		fmt.Println(channel)
		err := channel.Update()
		if err != nil {
			c.Error(err)
			return err
		}
		c.NoContent(http.StatusNoContent)
	} else {
		messageResponse(c, http.StatusBadRequest, "confirmがtrueではありません")
	}
	return nil
}

func values(m map[string]*ChannelForResponse) []*ChannelForResponse {
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
