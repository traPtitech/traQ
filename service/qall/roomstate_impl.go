package qall

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/traPtitech/traQ/event"
)

// Repository はRoomStateManagerインターフェースの実装
type Repository struct {
	LiveKitHost string
	APIKey      string
	APISecret   string
	RoomState   []RoomWithParticipants
	Hub         *hub.Hub
}

// GetState 現在のルーム状態を取得
func (r *Repository) GetState() []RoomWithParticipants {
	return r.RoomState
}

// InitializeRoomState LiveKit APIから現在のルーム状態を取得 (初期化時に利用)
func (r *Repository) InitializeRoomState() error {
	roomWithParticipants, err := r.GetRoomsWithParticipantsByLiveKitServer(context.Background())
	r.RoomState = roomWithParticipants
	return err
}

// AddParticipantToRoomState ルーム状態に参加者を追加
func (r *Repository) AddParticipantToRoomState(room *livekit.Room, participant *livekit.ParticipantInfo) {
	for i, roomState := range r.RoomState {
		if roomState.RoomID.String() == room.Name {
			t := time.Unix(participant.JoinedAt, 0).In(time.FixedZone("Asia/Tokyo", 9*60*60))
			r.RoomState[i].Participants = append(r.RoomState[i].Participants, Participant{
				Identity:   participant.Identity,
				JoinedAt:   t,
				Name:       participant.Name,
				Attributes: &participant.Attributes,
				CanPublish: participant.Permission.CanPublish,
			})

			if r.Hub != nil {
				r.Hub.Publish(hub.Message{
					Name: event.QallRoomStateChanged,
					Fields: hub.Fields{
						"roomStates": r.RoomState,
					},
				})
			}

			break
		}
	}
}

// UpdateParticipantCanPublish 参加者の発言権限を更新
func (r *Repository) UpdateParticipantCanPublish(roomID string, participantID string, canPublish bool) {
	for i, roomState := range r.RoomState {
		if roomState.RoomID.String() == roomID {
			for j, participant := range roomState.Participants {
				if participant.Identity == participantID {
					r.RoomState[i].Participants[j].CanPublish = canPublish

					if r.Hub != nil {
						r.Hub.Publish(hub.Message{
							Name: event.QallRoomStateChanged,
							Fields: hub.Fields{
								"roomStates": r.RoomState,
							},
						})
					}

					break
				}
			}
			break
		}
	}
}

// UpdateParticipant 参加者情報を更新
func (r *Repository) UpdateParticipant(roomID string, participant *livekit.ParticipantInfo) {
	for i, roomState := range r.RoomState {
		if roomState.RoomID.String() == roomID {
			for j, p := range roomState.Participants {
				if p.Identity == participant.Identity {
					t := time.Unix(participant.JoinedAt, 0).In(time.FixedZone("Asia/Tokyo", 9*60*60))
					r.RoomState[i].Participants[j] = Participant{
						Identity:   participant.Identity,
						JoinedAt:   t,
						Name:       participant.Name,
						Attributes: &participant.Attributes,
						CanPublish: participant.Permission.CanPublish,
					}

					if r.Hub != nil {
						r.Hub.Publish(hub.Message{
							Name: event.QallRoomStateChanged,
							Fields: hub.Fields{
								"roomStates": r.RoomState,
							},
						})
					}

					break
				}
			}
			break
		}
	}
}

// RemoveParticipant ルームから参加者を削除
func (r *Repository) RemoveParticipant(roomID string, participantID string) {
	for i, roomState := range r.RoomState {
		if roomState.RoomID.String() == roomID {
			for j, participant := range roomState.Participants {
				if participant.Identity == participantID {
					r.RoomState[i].Participants = slices.Delete(r.RoomState[i].Participants, j, j+1)

					if r.Hub != nil {
						r.Hub.Publish(hub.Message{
							Name: event.QallRoomStateChanged,
							Fields: hub.Fields{
								"roomStates": r.RoomState,
							},
						})
					}

					break
				}
			}
			break
		}
	}
}

// GetRoomsWithParticipantsByLiveKitServerAndSave LiveKitサーバーからルーム状態を取得して保存
func (r *Repository) GetRoomsWithParticipantsByLiveKitServerAndSave(ctx context.Context) error {
	roomWithParticipants, err := r.GetRoomsWithParticipantsByLiveKitServer(ctx)
	if err != nil {
		return err
	}
	r.RoomState = roomWithParticipants
	return nil
}

// AddRoomState ルーム状態を追加
func (r *Repository) AddRoomState(room RoomWithParticipants) {
	r.RoomState = append(r.RoomState, room)

	if r.Hub != nil {
		r.Hub.Publish(hub.Message{
			Name: event.QallRoomStateChanged,
			Fields: hub.Fields{
				"roomStates": r.RoomState,
			},
		})
	}
}

// UpdateRoomMetadata ルームのメタデータを更新
func (r *Repository) UpdateRoomMetadata(roomID string, metadata Metadata) {
	for i, roomState := range r.RoomState {
		if roomState.RoomID.String() == roomID {
			r.RoomState[i].Metadata = &metadata.Status

			if r.Hub != nil {
				r.Hub.Publish(hub.Message{
					Name: event.QallRoomStateChanged,
					Fields: hub.Fields{
						"roomStates": r.RoomState,
					},
				})
			}

			break
		}
	}
}

// RemoveRoomState ルーム状態を削除
func (r *Repository) RemoveRoomState(roomID string) {
	for i, roomState := range r.RoomState {
		if roomState.RoomID.String() == roomID {
			r.RoomState = append(r.RoomState[:i], r.RoomState[i+1:]...)

			if r.Hub != nil {
				r.Hub.Publish(hub.Message{
					Name: event.QallRoomStateChanged,
					Fields: hub.Fields{
						"roomStates": r.RoomState,
					},
				})
			}

			break
		}
	}
}

// NewLiveKitRoomServiceClient LiveKitルームサービスクライアントを作成
func (r *Repository) NewLiveKitRoomServiceClient() *lksdk.RoomServiceClient {
	return lksdk.NewRoomServiceClient(r.LiveKitHost, r.APIKey, r.APISecret)
}

// GetRoomsByLiveKitServer LiveKitサーバーからルーム一覧を取得
func (r *Repository) GetRoomsByLiveKitServer(ctx context.Context) (*livekit.ListRoomsResponse, error) {
	rsClient := r.NewLiveKitRoomServiceClient()
	return rsClient.ListRooms(ctx, &livekit.ListRoomsRequest{})
}

// GetParticipantsByLiveKitServer LiveKitサーバーから参加者一覧を取得
func (r *Repository) GetParticipantsByLiveKitServer(ctx context.Context, roomID string) (*livekit.ListParticipantsResponse, error) {
	rsClient := r.NewLiveKitRoomServiceClient()
	return rsClient.ListParticipants(ctx, &livekit.ListParticipantsRequest{
		Room: roomID,
	})
}

// GetRoomsWithParticipantsByLiveKitServer LiveKitサーバーからルーム状態と参加者一覧を取得
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
				Identity:   p.Identity,
				JoinedAt:   t,
				Name:       p.Name,
				Attributes: &p.Attributes,
			})
		}

		roomID, err := uuid.FromString(rm.Name)
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
			IsWebinar:    metadata.IsWebinar,
			RoomID:       roomID,
			Participants: Participants,
		})
	}

	return roomWithParticipants, nil
}

// GetRoomState ルーム状態を取得
func (r *Repository) GetRoomState(roomID string) *RoomWithParticipants {
	for i := range r.RoomState {
		if r.RoomState[i].RoomID.String() == roomID {
			return &r.RoomState[i]
		}
	}
	return nil
}

// NewRoomStateManager は新しいRoomStateManagerを作成する
func NewRoomStateManager(liveKitHost, apiKey, apiSecret string, hub *hub.Hub) RoomStateManager {
	return &Repository{
		LiveKitHost: liveKitHost,
		APIKey:      apiKey,
		APISecret:   apiSecret,
		RoomState:   make([]RoomWithParticipants, 0),
		Hub:         hub,
	}
}
