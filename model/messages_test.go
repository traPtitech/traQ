package model

import (
	"testing"
)

func TestCreateMessage(t *testing.T) {
	beforeTest(t)

	message := *makeMessage()
	copy := message
	if err := message.Create(); err != nil {
		t.Fatalf("Create method returns an error: %v", err)
	}

	has, err := db.ID(message.ID).Get(&message)
	if has != true {
		t.Error("Cannot find message in database")
	}
	if err != nil {
		t.Errorf("Failed to get message inserts before: %v", err)
	}

	if message.UserID != copy.UserID {
		t.Errorf("message.UserID is changed: before: %v, after: %v", copy.UserID, message.UserID)
	}
	if message.Text != copy.Text {
		t.Errorf("message.Text is changed: before: %v, after: %v", copy.Text, message.Text)
	}

	if message.CreatedAt == "" {
		t.Error("message.CreatedAt is not updated")
	}
	if message.UpdaterID == "" {
		t.Error("message.UpdaterID is not updated")
	}
}

func TestUpdateMessage(t *testing.T) {
	beforeTest(t)

	message := makeMessage()
	if err := message.Create(); err != nil {
		t.Fatalf("Create method returns an error: %v", err)
	}

	text := "nanachi"

	message.Text = text
	message.IsShared = false

	if err := message.Update(); err != nil {
		t.Errorf("Update method return an error: %v", err)
	}

	if message.Text != text {
		t.Error("message.Text is not updated")
	}
	if message.IsShared != false {
		t.Error("message.isShared is not updated")
	}
}

func TestGetMessagesFromChannel(t *testing.T) {
	beforeTest(t)

	channelID := CreateUUID()
	messages := makeChannelMessages(channelID)

	for i := 0; i < 10; i++ {
		if err := messages[i].Create(); err != nil {
			t.Fatalf("Create method returns an error: %v", err)
		}
	}

	res, err := GetMessagesFromChannel(channelID, 0, 0)
	if err != nil {
		t.Errorf("GetMessageFromChannel method returns an error: %v", err)
	}

	if len(res) != len(messages) {
		t.Errorf("Missing some of channel messages: want: %d, actual: %d", len(messages), len(res))
	}

	for i := 0; i < 10; i++ {
		if messages[9-i].ID != res[i].ID {
			t.Error("message is not ordered by createdAt")
		}
	}

	res2, err := GetMessagesFromChannel(channelID, 3, 5)
	if err != nil {
		t.Errorf("GetMessageFromChannel method returns an error: %v", err)
	}

	if len(res2) != 3 {
		t.Errorf("Missing some of channel messages: want: 3, actual: %d", len(res2))
	}

	if res2[0].ID != messages[4].ID {
		t.Error("message is not ordered by createdAt")
	}

}

func TestGetMessage(t *testing.T) {
	beforeTest(t)

	message := makeMessage()
	if err := message.Create(); err != nil {
		t.Fatalf("Create method returns an error: %v", err)
	}

	var r *Message
	r, err := GetMessage(message.ID)
	if err != nil {
		t.Errorf("GetMessage method returns an error: %v", err)
	}

	if r.Text != message.Text {
		t.Errorf("message.Text is changed: before: %v, after: %v", message.Text, r.Text)
	}
}
