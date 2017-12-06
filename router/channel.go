package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/traPtitech/traQ/model"
)

type ChannelForResponse struct {
	ChannelId  string
	Name       string
	Parent     string
	Children   []string
	Visibility bool
}

type ErrorResponse struct {
	Message string
}

type PostChannel struct {
	ChannelType string   `json:"type"`
	Member      []string `json:"member"`
	Name        string   `json:"name"`
	Parent      string   `json:"parent"`
}

func GetChannelsHandler(c echo.Context) error {
	sess, err := session.Get("sessions", c)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "セッションの取得に失敗しました")
		return fmt.Errorf("Failed to get session: %v", err)
	}

	var userId string
	if sess.Values["userId"] != nil {
		userId = sess.Values["userId"].(string)
	}
	channelList, err := model.GetChannelList(userId)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "チャンネルリストの取得中にエラーが発生しました")
		return fmt.Errorf("Failed to get channel list: %v", err)
	}

	response := make(map[string]*ChannelForResponse)

	for _, ch := range channelList {
		if response[ch.Id] == nil {
			response[ch.Id] = new(ChannelForResponse)
		}
		response[ch.Id].ChannelId = ch.Id
		response[ch.Id].Name = ch.Name
		response[ch.Id].Parent = ch.ParentId
		response[ch.Id].Visibility = !ch.IsHidden

		if response[ch.ParentId] == nil {
			response[ch.ParentId] = new(ChannelForResponse)
		}
		response[ch.ParentId].Children = append(response[ch.ParentId].Children, ch.Id)
	}

	c.JSON(http.StatusOK, values(response))
	return nil
}

func PostChannelsHandler(c echo.Context) error {
	// TODO: 同名・同階層のチャンネルのチェック
	sess, err := session.Get("sessions", c)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "セッションの取得に失敗しました")
		return fmt.Errorf("Failed to get session: %v", err)
	}

	var userId string
	if sess.Values["userId"] != nil {
		userId = sess.Values["userId"].(string)
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

	newChannel := new(model.Channels)
	newChannel.CreatorId = userId
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
			usersPrivateChannels := new(model.UsersPrivateChannels)
			usersPrivateChannels.ChannelId = newChannel.Id
			usersPrivateChannels.UserId = user
			err := usersPrivateChannels.Create()
			if err != nil {
				c.Error(err)
				return err
			}
		}
	}

	ch, err := model.GetChannelById(userId, newChannel.Id)

	if err != nil {
		c.Error(err)
		return err
	}

	c.JSON(http.StatusCreated, ch)
	return nil
}

func GetChannelsByChannelIdHandler(c echo.Context) error {
	sess, err := session.Get("sessions", c)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "セッションの取得に失敗しました")
		return fmt.Errorf("Failed to get session: %v", err)
	}

	var userId string
	if sess.Values["userId"] != nil {
		userId = sess.Values["userId"].(string)
	}

	channelId := c.Param("channelId")

	ok, err := model.ExistsChannel(channelId)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "チャンネルの検証中にサーバー内でエラーが発生しました")
		return err
	}
	if !ok {
		messageResponse(c, http.StatusNotFound, "指定されたチャンネルは存在しません")
		return nil
	}

	channel, err := model.GetChannelById(userId, channelId)
	if err != nil {
		c.Error(err)
		return fmt.Errorf("Failed to get channel: %v", err)
	}

	childrenIdList, err := model.GetChildrenChannelIdList(userId, channel.Id)
	if err != nil {
		c.Error(err)
		return fmt.Errorf("Failed to get children channel id list: %v", err)
	}

	response := ChannelForResponse{
		ChannelId:  channel.Id,
		Name:       channel.Name,
		Parent:     channel.ParentId,
		Visibility: !channel.IsHidden,
		Children:   childrenIdList,
	}

	c.JSON(http.StatusOK, response)
	return nil
}

func PutChannelsByChannelIdHandler(c echo.Context) error {
	// CHECK: 権限周り
	// TODO: 必要な引数があるかチェック
	sess, err := session.Get("sessions", c)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "セッションの取得に失敗しました")
		return fmt.Errorf("Failed to get session: %v", err)
	}

	var userId string
	if sess.Values["userId"] != nil {
		userId = sess.Values["userId"].(string)
	}
	var requestBody PostChannel
	c.Bind(&requestBody)

	channelId := c.Param("channelId")
	ok, err := model.ExistsChannel(channelId)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "チャンネルの検証中にサーバー内でエラーが発生しました")
		return err
	}
	if !ok {
		messageResponse(c, http.StatusNotFound, "指定されたチャンネルは存在しません")
		return nil
	}

	channel, err := model.GetChannelById(userId, channelId)
	if err != nil {
		c.Error(err)
		return fmt.Errorf("Failed to get channel: %v", err)
	}

	channel.Name = requestBody.Name
	channel.UpdaterId = userId

	if err := channel.Update(); err != nil {
		c.Error(err)
		return err
	}

	childrenIdList, err := model.GetChildrenChannelIdList(userId, channel.Id)
	if err != nil {
		c.Error(err)
		return fmt.Errorf("Failed to get children channel id list: %v", err)
	}

	response := ChannelForResponse{
		ChannelId:  channel.Id,
		Name:       channel.Name,
		Parent:     channel.ParentId,
		Visibility: !channel.IsHidden,
		Children:   childrenIdList,
	}

	c.JSON(http.StatusOK, response)
	return nil
}

func DeleteChannelsByChannelIdHandler(c echo.Context) error {
	// CHECK: 権限周り
	sess, err := session.Get("sessions", c)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "セッションの取得に失敗しました")
		return fmt.Errorf("Failed to get session: %v", err)
	}

	var userId string
	if sess.Values["userId"] != nil {
		userId = sess.Values["userId"].(string)
	}
	type confirm struct {
		Confirm bool `json:"confirm"`
	}
	var requestBody confirm
	c.Bind(&requestBody)

	channelId := c.Param("channelId")
	ok, err := model.ExistsChannel(channelId)
	if err != nil {
		messageResponse(c, http.StatusInternalServerError, "チャンネルの検証中にサーバー内でエラーが発生しました")
		return err
	}
	if !ok {
		messageResponse(c, http.StatusNotFound, "指定されたチャンネルは存在しません")
		return nil
	}

	channel, err := model.GetChannelById(userId, channelId)
	if err != nil {
		c.Error(err)
		return fmt.Errorf("Failed to get channel: %v", err)
	}

	if requestBody.Confirm {
		channel.UpdaterId = userId
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
