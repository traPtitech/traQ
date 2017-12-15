package model

import (
	"testing"
)

func TestCreateMessage(t *testing.T) {
	BeforeTest(t)
	defer Close()

	message := generateMessage()
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
	BeforeTest(t)
	defer Close()

	message := generateMessage()
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
	BeforeTest(t)
	defer Close()

	channelID := CreateUUID()
	messages := generateChannelMessages(channelID)

	for i := 0; i < 10; i++ {
		if err := messages[i].Create(); err != nil {
			t.Fatalf("Create method returns an error: %v", err)
		}
	}

	r, err := GetMessagesFromChannel(channelID)
	if err != nil {
		t.Errorf("GetMessageFromChannel method returns an error: %v", err)
	}

	if len(r) != len(messages) {
		t.Errorf("Missing some of channel messages: want: %d, actual: %d", len(messages), len(r))
	}

	for i := 0; i < 10; i++ {
		if messages[i].ID != r[i].ID {
			t.Error("message is not ordered by createdAt")
		}
	}

}

func TestGetMessage(t *testing.T) {
	BeforeTest(t)
	defer Close()

	message := generateMessage()
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

func generateMessage() Message {
	message := Message{
		UserID:    CreateUUID(),
		ChannelID: CreateUUID(),
		Text:      "テスト/is/popo",
		IsShared:  true,
	}
	return message
}

func generateChannelMessages(channelID string) []*Message {
	var messages [10]*Message

	for i := 0; i < 10; i++ {
		tmp := generateMessage()
		messages[i] = &tmp
		messages[i].ChannelID = channelID
	}

	return messages[:]
}
