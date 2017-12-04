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

type postChannel struct {
	channelType string   `json:"type"`
	member      []string `json:"member"`
	name        string   `json:"name"`
	parent      string   `json:"parent"`
}

func GetChannelsHandler(c echo.Context) error {
	sess, err := session.Get("sessions", c)
	if err != nil {
		return fmt.Errorf("Failed to get session: %v", err)
	}

	var userId string
	if sess.Values["userId"] != nil {
		userId = sess.Values["userId"].(string)
	}
	channelList, err := model.GetChannelList(userId)
	if err != nil {
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
	sess, err := session.Get("sessions", c)
	if err != nil {
		return fmt.Errorf("Failed to get session: %v", err)
	}

	var userId string
	if sess.Values["userId"] != nil {
		userId = sess.Values["userId"].(string)
	}
	var requestBody postChannel
	c.Bind(&requestBody)

	newChannel := new(model.Channels)
	newChannel.CreatorId = userId
	newChannel.Name = requestBody.name

	if requestBody.channelType == "public" {
		newChannel.IsPublic = true
	} else {
		newChannel.IsPublic = false
	}

	err := newChannel.Create()
	if err != nil {
		c.Error(err)
		return err
	}

	if requestBody.channelType == "public" {
		// TODO:通知周りの実装
	} else {
		for _, user := range requestBody.member {
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

	ch := model.GetChannelById(newChannel.Id)

	c.JSON(http.StatusCreated, ch)
	return nil
}

func GetChannelsByChannelIdHandler() {

}

func PutChannelsByChannelIdHandler() {
}

func DeleteChannelsByChannelIdHandler() {
}

func values(m map[string]*ChannelForResponse) []*ChannelForResponse {
	arr := []*ChannelForResponse{}
	for _, v := range m {
		arr = append(arr, v)
	}
	return arr
}
