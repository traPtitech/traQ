package model

import (
	"testing"
)

var testUserID = "403807a5-cae6-453e-8a09-fc75d5b4ca91"

func TestCreate(t *testing.T) {
	BeforeTest(t)
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
	BeforeTest(t)
	pin, err := makePinDetail("channelId", "messageId", testUserID)

	if err != nil {
		t.Fatal("Fail to create pin")
	}
	pinnedMessage, err := pin.GetPin("testChannelId")

	if err != nil {
		t.Fatal("Fail to get pinnedMessage")
	}

	if pinnedMessage.MessageId != "testMessageId" {
		t.Error("error")
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
