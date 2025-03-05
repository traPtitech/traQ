package qall

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/livekit/protocol/livekit"
	"github.com/stretchr/testify/assert"
)

func TestRepository_AddRoomState(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()
		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{},
		}

		// テスト用のルーム状態を準備
		roomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID:       roomID,
			Participants: []Participant{},
		}

		// テスト実行
		repo.AddRoomState(roomState)

		// 検証
		assert.Equal(t, 1, len(repo.RoomState))
		assert.Equal(t, roomID, repo.RoomState[0].RoomID)
	})
}

func TestRepository_RemoveRoomState(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()

		// テスト用のルーム状態を準備
		roomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID:       roomID,
			Participants: []Participant{},
		}

		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{roomState},
		}

		// テスト実行
		repo.RemoveRoomState(roomID.String())

		// 検証
		assert.Equal(t, 0, len(repo.RoomState))
	})

	t.Run("room_not_found", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()

		roomID := uuid.Must(uuid.NewV7())
		otherRoomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID:       roomID,
			Participants: []Participant{},
		}

		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{roomState},
		}

		// テスト実行
		repo.RemoveRoomState(otherRoomID.String())

		// 検証 - 何も変わらないはず
		assert.Equal(t, 1, len(repo.RoomState))
		assert.Equal(t, roomID, repo.RoomState[0].RoomID)
	})
}

func TestRepository_AddParticipantToRoomState(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()

		// テスト用のルーム状態を準備
		roomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID:       roomID,
			Participants: []Participant{},
		}

		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{roomState},
		}

		// LiveKitのルームとパーティシパント情報を準備
		room := &livekit.Room{
			Name: roomID.String(),
		}

		participantID := "user1"
		participantName := "Test User"
		canPublish := true

		participant := &livekit.ParticipantInfo{
			Identity: participantID,
			Name:     participantName,
			JoinedAt: time.Now().Unix(),
			Permission: &livekit.ParticipantPermission{
				CanPublish: canPublish,
			},
		}

		// テスト実行
		repo.AddParticipantToRoomState(room, participant)

		// 検証
		assert.Equal(t, 1, len(repo.RoomState))
		assert.Equal(t, 1, len(repo.RoomState[0].Participants))
		assert.Equal(t, participantID, *repo.RoomState[0].Participants[0].Identity)
		assert.Equal(t, participantName, *repo.RoomState[0].Participants[0].Name)
		assert.Equal(t, canPublish, *repo.RoomState[0].Participants[0].CanPublish)
	})

	t.Run("room_not_found", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()

		roomID := uuid.Must(uuid.NewV7())
		otherRoomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID:       roomID,
			Participants: []Participant{},
		}

		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{roomState},
		}

		// LiveKitのルームとパーティシパント情報を準備
		room := &livekit.Room{
			Name: otherRoomID.String(),
		}

		participant := &livekit.ParticipantInfo{
			Identity: "user1",
			Name:     "Test User",
		}

		// テスト実行
		repo.AddParticipantToRoomState(room, participant)

		// 検証 - 何も変わらないはず
		assert.Equal(t, 1, len(repo.RoomState))
		assert.Equal(t, 0, len(repo.RoomState[0].Participants))
	})
}

func TestRepository_RemoveParticipant(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()

		// 参加者情報を準備
		participantID := "user1"
		participantName := "Test User"
		joinedAt := time.Now()
		canPublish := true

		// テスト用のルーム状態を準備
		roomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID: roomID,
			Participants: []Participant{
				{
					Identity:   &participantID,
					Name:       &participantName,
					JoinedAt:   &joinedAt,
					CanPublish: &canPublish,
				},
			},
		}

		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{roomState},
		}

		// テスト実行
		repo.RemoveParticipant(roomID.String(), participantID)

		// 検証
		assert.Equal(t, 1, len(repo.RoomState))
		assert.Equal(t, 0, len(repo.RoomState[0].Participants))
	})

	t.Run("room_not_found", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()

		participantID := "user1"
		participantName := "Test User"
		joinedAt := time.Now()
		canPublish := true

		roomID := uuid.Must(uuid.NewV7())
		otherRoomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID: roomID,
			Participants: []Participant{
				{
					Identity:   &participantID,
					Name:       &participantName,
					JoinedAt:   &joinedAt,
					CanPublish: &canPublish,
				},
			},
		}

		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{roomState},
		}

		// テスト実行
		repo.RemoveParticipant(otherRoomID.String(), participantID)

		// 検証 - 何も変わらないはず
		assert.Equal(t, 1, len(repo.RoomState))
		assert.Equal(t, 1, len(repo.RoomState[0].Participants))
	})

	t.Run("participant_not_found", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()

		participantID := "user1"
		participantName := "Test User"
		joinedAt := time.Now()
		canPublish := true

		roomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID: roomID,
			Participants: []Participant{
				{
					Identity:   &participantID,
					Name:       &participantName,
					JoinedAt:   &joinedAt,
					CanPublish: &canPublish,
				},
			},
		}

		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{roomState},
		}

		// テスト実行
		repo.RemoveParticipant(roomID.String(), "nonexistent_user")

		// 検証 - 何も変わらないはず
		assert.Equal(t, 1, len(repo.RoomState))
		assert.Equal(t, 1, len(repo.RoomState[0].Participants))
	})
}

func TestRepository_UpdateParticipant(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()

		// 参加者情報を準備
		participantID := "user1"
		oldParticipantName := "Old Name"
		newParticipantName := "New Name"
		joinedAt := time.Now()
		canPublish := true

		// テスト用のルーム状態を準備
		roomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID: roomID,
			Participants: []Participant{
				{
					Identity:   &participantID,
					Name:       &oldParticipantName,
					JoinedAt:   &joinedAt,
					CanPublish: &canPublish,
				},
			},
		}

		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{roomState},
		}

		// 更新する参加者情報
		now := time.Now()
		participant := &livekit.ParticipantInfo{
			Identity: participantID,
			Name:     newParticipantName,
			JoinedAt: now.Unix(),
			Permission: &livekit.ParticipantPermission{
				CanPublish: canPublish,
			},
		}

		// テスト実行
		repo.UpdateParticipant(roomID.String(), participant)

		// 検証
		assert.Equal(t, 1, len(repo.RoomState))
		assert.Equal(t, 1, len(repo.RoomState[0].Participants))
		assert.Equal(t, participantID, *repo.RoomState[0].Participants[0].Identity)
		assert.Equal(t, newParticipantName, *repo.RoomState[0].Participants[0].Name)
	})

	t.Run("room_not_found", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()

		participantID := "user1"
		participantName := "Test User"
		joinedAt := time.Now()
		canPublish := true

		roomID := uuid.Must(uuid.NewV7())
		otherRoomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID: roomID,
			Participants: []Participant{
				{
					Identity:   &participantID,
					Name:       &participantName,
					JoinedAt:   &joinedAt,
					CanPublish: &canPublish,
				},
			},
		}

		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{roomState},
		}

		// 更新する参加者情報
		participant := &livekit.ParticipantInfo{
			Identity: participantID,
			Name:     "New Name",
		}

		// テスト実行
		repo.UpdateParticipant(otherRoomID.String(), participant)

		// 検証 - 何も変わらないはず
		assert.Equal(t, 1, len(repo.RoomState))
		assert.Equal(t, participantName, *repo.RoomState[0].Participants[0].Name)
	})

	t.Run("participant_not_found", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()

		participantID := "user1"
		participantName := "Test User"
		joinedAt := time.Now()
		canPublish := true

		roomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID: roomID,
			Participants: []Participant{
				{
					Identity:   &participantID,
					Name:       &participantName,
					JoinedAt:   &joinedAt,
					CanPublish: &canPublish,
				},
			},
		}

		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{roomState},
		}

		// 更新する参加者情報
		participant := &livekit.ParticipantInfo{
			Identity: "nonexistent_user",
			Name:     "New Name",
		}

		// テスト実行
		repo.UpdateParticipant(roomID.String(), participant)

		// 検証 - 何も変わらないはず
		assert.Equal(t, 1, len(repo.RoomState))
		assert.Equal(t, 1, len(repo.RoomState[0].Participants))
		assert.Equal(t, participantID, *repo.RoomState[0].Participants[0].Identity)
	})
}

func TestRepository_UpdateParticipantCanPublish(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()

		// 参加者情報を準備
		participantID := "user1"
		participantName := "Test User"
		joinedAt := time.Now()
		canPublish := true

		// テスト用のルーム状態を準備
		roomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID: roomID,
			Participants: []Participant{
				{
					Identity:   &participantID,
					Name:       &participantName,
					JoinedAt:   &joinedAt,
					CanPublish: &canPublish,
				},
			},
		}

		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{roomState},
		}

		// 参加者の発言権限を更新（falseに変更）
		newCanPublish := false

		// テスト実行
		repo.UpdateParticipantCanPublish(roomID.String(), participantID, newCanPublish)

		// 検証
		assert.Equal(t, 1, len(repo.RoomState))
		assert.Equal(t, 1, len(repo.RoomState[0].Participants))
		assert.Equal(t, participantID, *repo.RoomState[0].Participants[0].Identity)
		assert.Equal(t, newCanPublish, *repo.RoomState[0].Participants[0].CanPublish)
	})

	t.Run("room_not_found", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()

		participantID := "user1"
		participantName := "Test User"
		joinedAt := time.Now()
		canPublish := true

		roomID := uuid.Must(uuid.NewV7())
		otherRoomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID: roomID,
			Participants: []Participant{
				{
					Identity:   &participantID,
					Name:       &participantName,
					JoinedAt:   &joinedAt,
					CanPublish: &canPublish,
				},
			},
		}

		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{roomState},
		}

		// テスト実行
		repo.UpdateParticipantCanPublish(otherRoomID.String(), participantID, false)

		// 検証 - 何も変わらないはず
		assert.Equal(t, 1, len(repo.RoomState))
		assert.Equal(t, canPublish, *repo.RoomState[0].Participants[0].CanPublish)
	})

	t.Run("participant_not_found", func(t *testing.T) {
		t.Parallel()
		// モックセットアップ
		h := hub.New()

		participantID := "user1"
		participantName := "Test User"
		joinedAt := time.Now()
		canPublish := true

		roomID := uuid.Must(uuid.NewV7())
		roomState := RoomWithParticipants{
			RoomID: roomID,
			Participants: []Participant{
				{
					Identity:   &participantID,
					Name:       &participantName,
					JoinedAt:   &joinedAt,
					CanPublish: &canPublish,
				},
			},
		}

		repo := &Repository{
			Hub:       h,
			RoomState: []RoomWithParticipants{roomState},
		}

		// テスト実行
		repo.UpdateParticipantCanPublish(roomID.String(), "nonexistent_user", false)

		// 検証 - 何も変わらないはず
		assert.Equal(t, 1, len(repo.RoomState))
		assert.Equal(t, canPublish, *repo.RoomState[0].Participants[0].CanPublish)
	})
}

func TestRepository_GetState(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		// テスト用のルーム状態を準備
		roomID1 := uuid.Must(uuid.NewV7())
		roomID2 := uuid.Must(uuid.NewV7())

		roomStates := []RoomWithParticipants{
			{
				RoomID:       roomID1,
				Participants: []Participant{},
			},
			{
				RoomID:       roomID2,
				Participants: []Participant{},
			},
		}

		repo := &Repository{
			RoomState: roomStates,
		}

		// テスト実行
		result := repo.GetState()

		// 検証
		assert.Equal(t, 2, len(result))
		assert.Equal(t, roomID1, result[0].RoomID)
		assert.Equal(t, roomID2, result[1].RoomID)
	})

	t.Run("empty_state", func(t *testing.T) {
		t.Parallel()
		repo := &Repository{
			RoomState: []RoomWithParticipants{},
		}

		// テスト実行
		result := repo.GetState()

		// 検証
		assert.Equal(t, 0, len(result))
	})
}

func TestNewRoomStateManager(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		// テストデータを準備
		liveKitHost := "https://livekit.example.com"
		apiKey := "testAPIKey"
		apiSecret := "testAPISecret"
		h := hub.New()

		// テスト実行
		manager := NewRoomStateManager(liveKitHost, apiKey, apiSecret, h)

		// 型アサーションを行い、適切な型が返されていることを確認
		repo, ok := manager.(*Repository)
		assert.True(t, ok, "返されたマネージャーが*Repositoryではありません")

		// キャストしてフィールドを検証
		assert.Equal(t, liveKitHost, repo.LiveKitHost)
		assert.Equal(t, apiKey, repo.APIKey)
		assert.Equal(t, apiSecret, repo.APISecret)
		assert.Equal(t, 0, len(repo.RoomState))
		assert.Equal(t, h, repo.Hub)
	})
}

// LiveKit APIと通信するメソッドのテストはスキップ
func TestRepository_GetRoomsWithParticipantsByLiveKitServer(t *testing.T) {
	t.Skip("このテストは実際のLiveKitサーバーとの通信が必要です")
}

func TestRepository_InitializeRoomState(t *testing.T) {
	t.Skip("このテストは実際のLiveKitサーバーとの通信が必要です")
}
