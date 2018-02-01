package model

import "testing"

func TestTableNameClip(t *testing.T) {
	clip := &Clip{}
	if "clips" != clip.TableName() {
		t.Fatalf("tablename is wrong:want clips,actual %s", clip.TableName())
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
	messageCount := 5
	message := makeMessage()
	clip := &Clip{
		UserID:    testUserID,
		MessageID: message.ID,
	}

	if err := clip.Create(); err != nil {
		t.Fatalf("clip create failed: %v", err)
	}
	for i := 1; i < messageCount; i++ {
		mes := makeMessage()
		c := &Clip{
			UserID:    testUserID,
			MessageID: mes.ID,
		}

		if err := c.Create(); err != nil {
			t.Fatalf("clip create failed: %v", err)
		}
	}

	messages, err := GetClipedMessages(testUserID)
	if err != nil {
		t.Fatalf("getting cliped messages failed: %v", err)
	}

	if len(messages) != messageCount {
		t.Fatalf("messages count wrong: want %d, actual %d", messageCount, len(messages))
	}

	if messages[0].Text != message.Text {
		t.Fatalf("massage text wrong: want %s, actual %s", message.Text, messages[0].Text)
	}

}

func TestDeleteClip(t *testing.T) {
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
		t.Fatalf("messages count wrong: want %d, actual %d", messageCount-1, len(messages))
	}
}
