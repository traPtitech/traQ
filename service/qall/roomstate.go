package qall

import (
	"context"
	"encoding/json"
	"github.com/gofrs/uuid"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"time"
)

// Metadataに収容されるJSONの構造体
type Metadata struct {
	// ルームのメタデータ
	Status string `json:"status"`

	// webinarかどうか
	IsWebinar bool `json:"isWebinar"`
}

// Participant ルーム内の参加者一覧
type Participant struct {
	// Attributes ユーザーに関連付けられたカスタム属性
	Attributes *map[string]string `json:"attributes,omitempty"`

	// CanPublish 発言権限
	CanPublish *bool `json:"canPublish,omitempty"`

	// Identity ユーザーID_RandomUUID
	Identity *string `json:"identity,omitempty"`

	// JoinedAt 参加した時刻
	JoinedAt *time.Time `json:"joinedAt,omitempty"`

	// Name 表示名
	Name *string `json:"name,omitempty"`
}

// RoomWithParticipants defines model for RoomWithParticipants.
type RoomWithParticipants struct {
	// IsWebinar ウェビナールームかどうか
	IsWebinar *bool `json:"isWebinar,omitempty"`

	// Metadata ルームに関連付けられたカスタム属性
	Metadata     *string       `json:"metadata,omitempty"`
	Participants []Participant `json:"participants"`

	// RoomId ルームのID
	RoomId uuid.UUID `json:"roomId"`
}

type Repository struct {
	LiveKitHost string
	ApiKey      string
	ApiSecret   string
	RoomState   []RoomWithParticipants
}

// InitializeRoomState LiveKit APIから現在のルーム状態を取得 (初期化時に利用)
func (r *Repository) InitializeRoomState() error {
	roomWithParticipants, err := r.GetRoomsWithParticipantsByLiveKitServer(context.Background())
	r.RoomState = roomWithParticipants
	return err
}

func (r *Repository) AddParticipantToRoomState(room *livekit.Room, participant *livekit.ParticipantInfo) {
	for i, roomState := range r.RoomState {
		if roomState.RoomId.String() == room.Name {
			t := time.Unix(participant.JoinedAt, 0).In(time.FixedZone("Asia/Tokyo", 9*60*60))
			r.RoomState[i].Participants = append(r.RoomState[i].Participants, Participant{
				Identity:   &participant.Identity,
				JoinedAt:   &t,
				Name:       &participant.Name,
				Attributes: &participant.Attributes,
				CanPublish: &participant.Permission.CanPublish,
			})
		}
	}
}

func (r *Repository) UpdateParticipantCanPublish(roomId string, participantId string, canPublish bool) {
	for i, roomState := range r.RoomState {
		if roomState.RoomId.String() == roomId {
			for j, participant := range roomState.Participants {
				if *participant.Identity == participantId {
					r.RoomState[i].Participants[j].CanPublish = &canPublish
				}
			}
		}
	}
}

func (r *Repository) UpdateParticipant(roomId string, participant *livekit.ParticipantInfo) {
	for i, roomState := range r.RoomState {
		if roomState.RoomId.String() == roomId {
			for j, p := range roomState.Participants {
				if *p.Identity == participant.Identity {
					t := time.Unix(participant.JoinedAt, 0).In(time.FixedZone("Asia/Tokyo", 9*60*60))
					r.RoomState[i].Participants[j] = Participant{
						Identity:   &participant.Identity,
						JoinedAt:   &t,
						Name:       &participant.Name,
						Attributes: &participant.Attributes,
					}
				}
			}
		}
	}
}

func (r *Repository) RemoveParticipant(roomId string, participantId string) {
	for i, roomState := range r.RoomState {
		if roomState.RoomId.String() == roomId {
			for j, participant := range roomState.Participants {
				if *participant.Identity == participantId {
					r.RoomState[i].Participants = append(r.RoomState[i].Participants[:j], r.RoomState[i].Participants[j+1:]...)
				}
			}
		}
	}
}

func (r *Repository) GetRoomsWithParticipantsByLiveKitServerAndSave(ctx context.Context) error {
	roomWithParticipants, err := r.GetRoomsWithParticipantsByLiveKitServer(ctx)
	if err != nil {
		return err
	}
	r.RoomState = roomWithParticipants
	return nil
}

func (r *Repository) AddRoomState(room RoomWithParticipants) {
	r.RoomState = append(r.RoomState, room)
}

func (r *Repository) CreateRoomState(roomId string) error {
	roomUUID, err := uuid.FromString(roomId)
	if err != nil {
		return err
	}
	r.AddRoomState(RoomWithParticipants{
		RoomId:       roomUUID,
		Participants: make([]Participant, 0),
	})
	return nil
}

func (r *Repository) RemoveRoomState(roomId string) {
	for i, roomState := range r.RoomState {
		if roomState.RoomId.String() == roomId {
			r.RoomState = append(r.RoomState[:i], r.RoomState[i+1:]...)
		}
	}
}

func (r *Repository) NewLiveKitRoomServiceClient() *lksdk.RoomServiceClient {
	return lksdk.NewRoomServiceClient(r.LiveKitHost, r.ApiKey, r.ApiSecret)
}

func (r *Repository) GetRoomsByLiveKitServer(ctx context.Context) (*livekit.ListRoomsResponse, error) {
	rsClient := r.NewLiveKitRoomServiceClient()
	return rsClient.ListRooms(ctx, &livekit.ListRoomsRequest{})
}

func (r *Repository) GetParticipantsByLiveKitServer(ctx context.Context, roomId string) (*livekit.ListParticipantsResponse, error) {
	rsClient := r.NewLiveKitRoomServiceClient()
	return rsClient.ListParticipants(ctx, &livekit.ListParticipantsRequest{
		Room: roomId,
	})
}

func (r *Repository) GetRoomsWithParticipantsByLiveKitServer(ctx context.Context) ([]RoomWithParticipants, error) {
	roomResp, err := r.GetRoomsByLiveKitServer(ctx)
	if err != nil {
		return nil, err
	}

	var roomWithParticipants []RoomWithParticipants
	for _, rm := range roomResp.Rooms {
		partResp, err := r.GetParticipantsByLiveKitServer(ctx, rm.Name)
		if err != nil {
			return nil, err
		}

		var Participants []Participant
		for _, p := range partResp.Participants {
			t := time.Unix(p.JoinedAt, 0).In(time.FixedZone("Asia/Tokyo", 9*60*60))
			Participants = append(Participants, Participant{
				Identity:   &p.Identity,
				JoinedAt:   &t,
				Name:       &p.Name,
				Attributes: &p.Attributes,
			})
		}

		roomId, err := uuid.FromString(rm.Name)
		if err != nil {
			return nil, err
		}

		var metadata *Metadata
		// rm.MetadataをJSON文字列としてunmarshalする
		err = json.Unmarshal([]byte(rm.Metadata), &metadata)
		if err != nil {
			return nil, err
		}

		roomWithParticipants = append(roomWithParticipants, RoomWithParticipants{
			Metadata:     &metadata.Status,
			IsWebinar:    &metadata.IsWebinar,
			RoomId:       roomId,
			Participants: Participants,
		})
	}

	return roomWithParticipants, nil
}
