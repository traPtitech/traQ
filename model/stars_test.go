package model

import (
	"strconv"
	"testing"
)

func TestTableNameStar(t *testing.T) {
	star := &Star{}
	if "stars" != star.TableName() {
		t.Fatalf("tablename is wrong:want stars, actual %s", star.TableName())
	}
}

func TestCreateStar(t *testing.T) {
	beforeTest(t)
	channel, err := makeChannelDetail(testUserID, "test", "", true)
	if err != nil {
		t.Fatalf("can't make channel: %v", err)
	}

	star := &Star{
		UserID:    testUserID,
		ChannelID: channel.ID,
	}

	if err := star.Create(); err != nil {
		t.Fatalf("star create failed: %v", err)
	}
}

func TestGetStaredChannels(t *testing.T) {
	beforeTest(t)
	channelCount := 5
	channel, err := makeChannelDetail(testUserID, "test0", "", true)

	if err != nil {
		t.Fatalf("can't make channel: %v", err)
	}

	star := &Star{
		UserID:    testUserID,
		ChannelID: channel.ID,
	}

	if err := star.Create(); err != nil {
		t.Fatalf("star create failed: %v", err)
	}

	for i := 1; i < channelCount; i++ {
		ch, err := makeChannelDetail(testUserID, "test"+strconv.Itoa(i), "", true)
		if err != nil {
			t.Fatalf("can't make channel: %v", err)
		}

		s := &Star{
			UserID:    testUserID,
			ChannelID: ch.ID,
		}

		if err := s.Create(); err != nil {
			t.Fatalf("star create failed: %v", err)
		}
	}

	channels, err := GetStaredChannels(testUserID)
	if err != nil {
		t.Fatalf("getting stared channels failed: %v", err)
	}

	if len(channels) != channelCount {
		t.Fatalf("channels count wrong: want %d, actual %d", channelCount, len(channels))
	}
}

func TestDeleteStar(t *testing.T) {
	beforeTest(t)
	channelCount := 5
	for i := 0; i < channelCount; i++ {
		ch, err := makeChannelDetail(testUserID, "test"+strconv.Itoa(i), "", true)
		if err != nil {
			t.Fatalf("can't make channel: %v", err)
		}

		s := &Star{
			UserID:    testUserID,
			ChannelID: ch.ID,
		}

		if err := s.Create(); err != nil {
			t.Fatalf("star create failed: %v", err)
		}
	}

	channels, err := GetStaredChannels(testUserID)
	if err != nil {
		t.Fatalf("getting stared channels failed: %v", err)
	}

	star := &Star{
		UserID:    testUserID,
		ChannelID: channels[0].ID,
	}
	if err := star.Delete(); err != nil {
		t.Fatalf("star delete failed: %v", err)
	}

	channels, err = GetStaredChannels(testUserID)
	if err != nil {
		t.Fatalf("getting stared channels failed: %v", err)
	}

	if len(channels) != channelCount-1 {
		t.Fatalf("channels count wrong: want %d, actual %d", channelCount, len(channels))
	}
}
