package model

import (
	"testing"
)

func TestTableNamePin(t *testing.T) {

	pin := &Pin{}
	if "pins" != pin.Tablename() {
		t.Fatalf("Tablename is wrong: want pins, actual %s", pin.Tablename())
	}
}

func TestCreatePins(t *testing.T) {
	beforeTest(t)
	message := makeMessage()
	pin := &Pin{
		UserID:    testUserID,
		MessageID: message.ID,
		ChannelID: message.ChannelID,
	}

	if err := pin.Create(); err != nil {
		t.Fatalf("Pin create faild: %v", err)
	}

}
func TestGetPinnedMessage(t *testing.T) {
	beforeTest(t)
	messageCount := 5
	message := makeMessage()
	pin := &Pin{
		UserID:    testUserID,
		MessageID: message.ID,
		ChannelID: message.ChannelID,
	}
	if err := pin.Create(); err != nil {
		t.Fatalf("pin create failed: %v", err)
	}
	for i := 0; i < messageCount; i++ {
		mes := makeMessage()
		p := &Pin{
			UserID:    testUserID,
			MessageID: mes.ID,
			ChannelID: mes.ChannelID,
		}
		if err := p.Create(); err != nil {
			t.Fatalf("Pin create is failed: %v", err)
		}
	}
	messages, err := GetPinMesssages(pin.ChannelID)

	if err != nil {
		t.Fatalf("Getting pined messages failed: %v", err)
	}

	if len(messages) != messageCount {
		print(messages)
		t.Fatalf("Message count wrong: want %d, actual %d", messageCount, len(messages))
	}

	if messages[0].Text != message.Text {
		t.Fatalf("Message text wrong: want %s, actual %s", message.Text, messages[0].Text)
	}
}
func TestDeletePin(t *testing.T) {
	beforeTest(t)
	messageCount := 5
	for i := 0; i < messageCount; i++ {
		message := makeMessage()
		pin := &Pin{
			UserID:    testUserID,
			MessageID: message.ID,
			ChannelID: message.ChannelID,
		}
		if err := pin.Create(); err != nil {
			t.Fatalf("Pin create faild: %v", err)
		}
	}

	message := makeMessage()

	messages, err := GetPinMesssages(message.ChannelID)

	if err != nil {
		t.Fatalf("Pin create fialed: %v", err)
	}

	pin := &Pin{
		UserID:    testUserID,
		MessageID: messages[0].ID,
		ChannelID: messages[0].ChannelID,
	}
	if err := pin.DeletePin(); err != nil {
		t.Fatalf("Pin delete failaed: %v", err)
	}

	messages, err = GetPinMesssages(messages[0].ChannelID)
	if err != nil {
		t.Fatalf("Pin cliate failed: %v", err)
	}

	if len(messages) != messageCount-1 {
		t.Fatalf("Message count wrong: want %d, actual %d", messageCount-1, len(messages))
	}
}
