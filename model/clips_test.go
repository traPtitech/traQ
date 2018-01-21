package model

import "testing"

func TestTableName(t *testing.T) {
	clip := &Clip{}
	if "clips" != clip.TableName() {
		t.Fatalf("tablename is worng:want clips,actual %s", clip.TableName())
	}
}

func TestCreateClip(t *testing.T) {
	beforeTest(t)
	message := makeMessage()
	clip := &Clip{
		UserID:    testUserID,
		MessageID: message.ID,
	}

	if err := clip.Create(); err != nil {
		t.Fatalf("clip create failed: %v", err)
	}
}

func TestGetClipedMessages(t *testing.T) {
	beforeTest(t)
	message := makeMessage()
	clip := &Clip{
		UserID:    testUserID,
		MessageID: message.ID,
	}

	if err := clip.Create(); err != nil {
		t.Fatalf("clip create failed: %v", err)
	}

	messages, err := GetClipedMessages(testUserID)
	if err != nil {
		t.Fatalf("clip create failed: %v", err)
	}

	if len(messages) != 1 {
		t.Fatalf("messages count wrong: want 1, actual %d", len(messages))
	}

	if messages[0].Text != message.Text {
		t.Fatalf("message text wrong: want %s, actual %s", message.Text, messages[0].Text)
	}
}

func TestDelete(t *testing.T) {
	beforeTest(t)
	messageCount := 5
	for i := 0; i < messageCount; i++ {
		message := makeMessage()
		clip := &Clip{
			UserID:    testUserID,
			MessageID: message.ID,
		}

		if err := clip.Create(); err != nil {
			t.Fatalf("clip create failed: %v", err)
		}
	}
	messages, err := GetClipedMessages(testUserID)
	if err != nil {
		t.Fatalf("clip create failed: %v", err)
	}

	if len(messages) != messageCount {
		t.Fatalf("messages count wrong: want 1, actual %d", len(messages))
	}

	clip := &Clip{
		UserID:    testUserID,
		MessageID: messages[0].ID,
	}
	if err := clip.Delete(); err != nil {
		t.Fatalf("clip delete failed: %v", err)
	}

	messages, err = GetClipedMessages(testUserID)
	if err != nil {
		t.Fatalf("clip create failed: %v", err)
	}

	if len(messages) != messageCount-1 {
		t.Fatalf("messages count wrong: want 1, actual %d", len(messages))
	}
}
