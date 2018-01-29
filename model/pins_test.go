package model

import (
	"testing"
)

func TestCreatePins(t *testing.T) {
	beforeTest(t)
	pin, err := makePinDetail("channelId", "messageId", testUserID)

	if err != nil {
		t.Error(err)
	}

	if pin.UserId != testUserID {
		t.Errorf("CreatorId : want %s, actual %s", testUserID, pin.UserId)
	}
	if pin.MessageId != "messageId" {
		t.Errorf("UserId : want %s, actual %s", "testPin", pin.MessageId)
	}
	if pin.ChannelId != "channelId" {
		t.Errorf("ChannelId : want %s, actual %s", "channelId", pin.MessageId)
	}

}

func TestGetPinnedMessage(t *testing.T) {
	beforeTest(t)

	pinnedMessage, err := GetPin("testChannelId")

	if err != nil {
		t.Fatal("Fail to get pinnedMessage")
	}

	if pinnedMessage.MessageId != "testMessageId" {
		t.Error("fail to get messageId")
	}
	if pinnedMessage.ChannelId != "testChannelId" {
		t.Error("fail to get testChannelId")
	}
	if pinnedMessage.UserId != testUserID {
		t.Error("fail to create testuserid")
	}
}

func makePin() error {
	pin := &Pins{}
	pin.UserId = testUserID
	pin.MessageId = "testMessageId"
	pin.ChannelId = "testChannelId"
	return pin.Create()
}

func makePinDetail(channelId, messageId, userId string) (*Pins, error) {
	pin := &Pins{}
	pin.ChannelId = channelId
	pin.MessageId = messageId
	pin.UserId = userId
	err := pin.Create()
	return pin, err
}
