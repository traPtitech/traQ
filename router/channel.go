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
