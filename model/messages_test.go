package model

import (
	"testing"
)

func TestCreateMessage(t *testing.T) {
	BeforeTest(t)
	defer Close()
	
	message := generateMessage()
	copy := message
	err := message.Create()
	if err != nil {
		t.Fatalf("Create method returns an error: %v", err)
	}
	
	has, err := db.ID(message.Id).Get(&message)
	if has != true {
		t.Error("Cannot find message in database")
	}
	if err != nil {
		t.Errorf("Failed to get message inserts before: %v", err)
	}

	if message.UserId != copy.UserId {
		t.Errorf("message.UserId is changed: before: %v, after: %v", copy.UserId, message.UserId)
	}
	if message.Text != copy.Text {
		t.Errorf("message.Text is changed: before: %v, after: %v", copy.Text, message.Text)
	}

	if message.CreatedAt == "" {
		t.Error("message.CreatedAt is not updated")
	}
	if message.UpdaterId == "" {
		t.Error("message.UpdaterId is not updated")
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
	
}

func TestGetMessage(t *testing.T) {
	
}

func TestDeleteMessage(t *testing.T) {
	
}

func generateMessage() Messages {
	message := new(Messages)
	message.UserId = CreateUUID()
	message.ChannelId = CreateUUID()
	message.Text = "テスト/is/popo" // TODO: randomな文字列
	message.IsShared = true
	return *message
}