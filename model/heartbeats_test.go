package model

import (
	"testing"
	"time"
)

const testChannelID = ""

func TestGetHeartbeatStatus(t *testing.T) {
	statusesMutex.Lock()
	HeartbeatStatuses[testChannelID] = &HeartbeatStatus{
		ChannelID: testChannelID,
		UserStatuses: []*UserStatus{
			{
				UserID:   testUserID,
				Status:   "editing",
				LastTime: time.Now(),
			},
		},
	}
	statusesMutex.Unlock()

	status, ok := GetHeartbeatStatus(testChannelID)

	if len(status.UserStatuses) != 1 {
		t.Fatalf("statuses length wrong: want 1, actual %d", len(status.UserStatuses))
	}

	status, ok = GetHeartbeatStatus(CreateUUID())

	if ok {
		t.Fatalf("ok is not false")
	}
}

func TestHeartbeat(t *testing.T) {
	tickTime = 10 * time.Millisecond
	timeoutDuration = -20 * time.Millisecond
	statusesMutex.Lock()
	HeartbeatStatuses[testChannelID] = &HeartbeatStatus{
		ChannelID: testChannelID,
		UserStatuses: []*UserStatus{
			{
				UserID:   testUserID,
				Status:   "editing",
				LastTime: time.Now(),
			},
		},
	}

	statusesMutex.Unlock()
	if len(HeartbeatStatuses[testChannelID].UserStatuses) != 1 {
		t.Fatalf("statuses length wrong: want 1, actual %d", len(HeartbeatStatuses[testChannelID].UserStatuses))
	}

	if err := HeartbeatStart(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(50 * time.Millisecond)

	statusesMutex.Lock()
	if len(HeartbeatStatuses[testChannelID].UserStatuses) != 0 {
		t.Fatalf("statuses length wrong: want 0, actual %d", len(HeartbeatStatuses[testChannelID].UserStatuses))
	}
	statusesMutex.Unlock()

	if err := HeartbeatStop(); err != nil {
		t.Fatal(err)
	}

}
