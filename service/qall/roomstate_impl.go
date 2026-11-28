package qall

import (
	"context"
	"encoding/json/v2"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/leandro-lugaresi/hub"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/traPtitech/traQ/event"
)

var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

// Repository はRoomStateManagerインターフェースの実装
//
// RoomState は LiveKit の Webhook と API リクエストから並行して更新されるため、
// 読み書きはすべて mu を通して行う。外へ返す値は内部状態と共有しないようにコピーする。
type Repository struct {
	LiveKitHost string
	APIKey      string
	APISecret   string
	RoomState   []RoomWithParticipants
	Hub         *hub.Hub

	mu sync.RWMutex
}

// GetState 現在のルーム状態を取得
func (r *Repository) GetState() []RoomWithParticipants {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneRoomStates(r.RoomState)
}

// InitializeRoomState LiveKit APIから現在のルーム状態を取得 (初期化時に利用)
//
// traQ はルーム状態をメモリ上にしか持たないため、再起動直後は LiveKit から読み直さないと
// 通話中のルームが空として扱われてしまう。
func (r *Repository) InitializeRoomState(ctx context.Context) error {
	return r.GetRoomsWithParticipantsByLiveKitServerAndSave(ctx)
}

// AddParticipantToRoomState ルーム状態に参加者を追加
//
// ルームが状態に存在しない場合 (traQ の再起動中に作成された場合など) はルームごと追加する。
// 同じ Identity の参加者が既にいる場合 (Webhook の再送など) は上書きする。
func (r *Repository) AddParticipantToRoomState(room *livekit.Room, participant *livekit.ParticipantInfo) {
	if room == nil || participant == nil {
		return
	}
	p := participantFromLiveKit(participant)

	r.mu.Lock()
	defer r.mu.Unlock()

	i := r.indexOfRoom(room.Name)
	if i < 0 {
		roomID, err := uuid.FromString(room.Name)
		if err != nil {
			return
		}
		r.RoomState = append(r.RoomState, RoomStateFromLiveKit(roomID, room.Metadata))
		i = len(r.RoomState) - 1
	}

	participants := r.RoomState[i].Participants
	if j := slices.IndexFunc(participants, func(q Participant) bool { return q.Identity == p.Identity }); j >= 0 {
		participants[j] = p
	} else {
		r.RoomState[i].Participants = append(participants, p)
	}
	r.publishLocked()
}

// UpdateParticipantCanPublish 参加者の発言権限を更新
func (r *Repository) UpdateParticipantCanPublish(roomID string, participantID string, canPublish bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	i := r.indexOfRoom(roomID)
	if i < 0 {
		return
	}
	for j, participant := range r.RoomState[i].Participants {
		if participant.Identity == participantID {
			r.RoomState[i].Participants[j].CanPublish = canPublish
			r.publishLocked()
			return
		}
	}
}

// UpdateParticipant 参加者情報を更新
func (r *Repository) UpdateParticipant(roomID string, participant *livekit.ParticipantInfo) {
	if participant == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	i := r.indexOfRoom(roomID)
	if i < 0 {
		return
	}
	for j, p := range r.RoomState[i].Participants {
		if p.Identity == participant.Identity {
			r.RoomState[i].Participants[j] = participantFromLiveKit(participant)
			r.publishLocked()
			return
		}
	}
}

// RemoveParticipant ルームから参加者を削除
func (r *Repository) RemoveParticipant(roomID string, participantID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	i := r.indexOfRoom(roomID)
	if i < 0 {
		return
	}
	for j, participant := range r.RoomState[i].Participants {
		if participant.Identity == participantID {
			r.RoomState[i].Participants = slices.Delete(r.RoomState[i].Participants, j, j+1)
			r.publishLocked()
			return
		}
	}
}

// GetRoomsWithParticipantsByLiveKitServerAndSave LiveKitサーバーからルーム状態を取得して保存
func (r *Repository) GetRoomsWithParticipantsByLiveKitServerAndSave(ctx context.Context) error {
	roomWithParticipants, err := r.GetRoomsWithParticipantsByLiveKitServer(ctx)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.RoomState = roomWithParticipants
	r.publishLocked()
	return nil
}

// AddRoomState ルーム状態を追加
//
// 既に同じルームがある場合は何もしない (参加者を消してしまわないように)
func (r *Repository) AddRoomState(room RoomWithParticipants) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.indexOfRoom(room.RoomID.String()) >= 0 {
		return
	}
	if room.Participants == nil {
		room.Participants = []Participant{}
	}
	r.RoomState = append(r.RoomState, room)
	r.publishLocked()
}

// UpdateRoomMetadata ルームのメタデータを更新
func (r *Repository) UpdateRoomMetadata(roomID string, metadata Metadata) {
	r.mu.Lock()
	defer r.mu.Unlock()

	i := r.indexOfRoom(roomID)
	if i < 0 {
		return
	}
	status := metadata.Status
	r.RoomState[i].Metadata = &status
	r.RoomState[i].Publishers = slices.Clone(metadata.Publishers)
	r.publishLocked()
}

// RemoveRoomState ルーム状態を削除
func (r *Repository) RemoveRoomState(roomID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	i := r.indexOfRoom(roomID)
	if i < 0 {
		return
	}
	r.RoomState = slices.Delete(r.RoomState, i, i+1)
	r.publishLocked()
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
//
// traQ のチャンネルIDではない名前のルームは無視する。
func (r *Repository) GetRoomsWithParticipantsByLiveKitServer(ctx context.Context) ([]RoomWithParticipants, error) {
	roomResp, err := r.GetRoomsByLiveKitServer(ctx)
	if err != nil {
		return nil, err
	}

	roomWithParticipants := make([]RoomWithParticipants, 0, len(roomResp.Rooms))
	for _, rm := range roomResp.Rooms {
		roomID, err := uuid.FromString(rm.Name)
		if err != nil {
			continue
		}

		partResp, err := r.GetParticipantsByLiveKitServer(ctx, rm.Name)
		if err != nil {
			return nil, err
		}

		participants := make([]Participant, 0, len(partResp.Participants))
		for _, p := range partResp.Participants {
			participants = append(participants, participantFromLiveKit(p))
		}

		room := RoomStateFromLiveKit(roomID, rm.Metadata)
		room.Participants = participants
		roomWithParticipants = append(roomWithParticipants, room)
	}

	return roomWithParticipants, nil
}

// GetRoomState ルーム状態を取得
//
// 返り値は内部状態のコピーなので、変更しても状態には反映されない。
func (r *Repository) GetRoomState(roomID string) *RoomWithParticipants {
	r.mu.RLock()
	defer r.mu.RUnlock()

	i := r.indexOfRoom(roomID)
	if i < 0 {
		return nil
	}
	room := cloneRoomState(r.RoomState[i])
	return &room
}

// indexOfRoom roomIDのルームの添字を返す。存在しなければ -1 (mu を取得した状態で呼ぶこと)
func (r *Repository) indexOfRoom(roomID string) int {
	return slices.IndexFunc(r.RoomState, func(room RoomWithParticipants) bool {
		return room.RoomID.String() == roomID
	})
}

// publishLocked 現在の状態を QallRoomStateChanged イベントとして配信する (mu を取得した状態で呼ぶこと)
func (r *Repository) publishLocked() {
	if r.Hub == nil {
		return
	}
	r.Hub.Publish(hub.Message{
		Name: event.QallRoomStateChanged,
		Fields: hub.Fields{
			"roomStates": cloneRoomStates(r.RoomState),
		},
	})
}

// participantFromLiveKit LiveKit の参加者情報を Participant に変換する
func participantFromLiveKit(p *livekit.ParticipantInfo) Participant {
	attributes := maps.Clone(p.Attributes)
	if attributes == nil {
		attributes = map[string]string{}
	}
	return Participant{
		Identity:   p.Identity,
		JoinedAt:   time.Unix(p.JoinedAt, 0).In(jst),
		Name:       p.Name,
		Attributes: &attributes,
		CanPublish: p.GetPermission().GetCanPublish(),
	}
}

// RoomStateFromLiveKit LiveKit のルームのメタデータから、参加者のいないルームの状態を作る
func RoomStateFromLiveKit(roomID uuid.UUID, rawMetadata string) RoomWithParticipants {
	metadata := ParseMetadata(rawMetadata)
	return RoomWithParticipants{
		IsWebinar:    metadata.IsWebinar,
		Metadata:     &metadata.Status,
		RoomID:       roomID,
		Participants: []Participant{},
		Publishers:   metadata.Publishers,
	}
}

// ParseMetadata ルームのメタデータ (JSON) を解釈する。空や不正な値の場合はゼロ値を返す
func ParseMetadata(raw string) Metadata {
	var metadata Metadata
	if raw == "" {
		return metadata
	}
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return Metadata{}
	}
	return metadata
}

func cloneRoomStates(rooms []RoomWithParticipants) []RoomWithParticipants {
	res := make([]RoomWithParticipants, len(rooms))
	for i, room := range rooms {
		res[i] = cloneRoomState(room)
	}
	return res
}

func cloneRoomState(room RoomWithParticipants) RoomWithParticipants {
	if room.Metadata != nil {
		metadata := *room.Metadata
		room.Metadata = &metadata
	}
	participants := make([]Participant, len(room.Participants))
	for i, p := range room.Participants {
		if p.Attributes != nil {
			attributes := maps.Clone(*p.Attributes)
			p.Attributes = &attributes
		}
		participants[i] = p
	}
	room.Participants = participants
	room.Publishers = slices.Clone(room.Publishers)
	return room
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
