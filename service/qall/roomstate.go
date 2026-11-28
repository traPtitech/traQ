package qall

import (
	"context"
	"slices"
	"time"

	"github.com/gofrs/uuid"
	"github.com/livekit/protocol/livekit"
)

// Metadataに収容されるJSONの構造体
type Metadata struct {
	// ルームのメタデータ
	Status string `json:"status"`

	// webinarかどうか
	IsWebinar bool `json:"isWebinar"`

	// Publishers ウェビナーで発言権限 (= 管理権限) を持つユーザーのID
	//
	// 参加者単位ではなくユーザー単位で持つので、抜けて入り直しても権限は残る。
	// LiveKit のルームのメタデータに保存されるので、traQ を再起動しても残る。
	Publishers []uuid.UUID `json:"publishers,omitempty"`
}

// IsPublisher ユーザーがウェビナーで発言権限 (= 管理権限) を持っているかどうか
func (m Metadata) IsPublisher(userID uuid.UUID) bool {
	return slices.Contains(m.Publishers, userID)
}

// Participant ルーム内の参加者一覧
type Participant struct {
	// Attributes ユーザーに関連付けられたカスタム属性
	Attributes *map[string]string `json:"attributes,omitempty"`

	// CanPublish 発言権限
	CanPublish bool `json:"canPublish"`

	// Identity ユーザーID_RandomUUID
	Identity string `json:"identity"`

	// JoinedAt 参加した時刻
	JoinedAt time.Time `json:"joinedAt"`

	// Name 表示名
	Name string `json:"name"`
}

// RoomMetadata ルームの状態から LiveKit のルームに保存するメタデータを作る
func (r RoomWithParticipants) RoomMetadata() Metadata {
	status := ""
	if r.Metadata != nil {
		status = *r.Metadata
	}
	return Metadata{
		Status:     status,
		IsWebinar:  r.IsWebinar,
		Publishers: slices.Clone(r.Publishers),
	}
}

// CanPublish ユーザーがこのルームで発言できるかどうか
//
// ウェビナーでなければ誰でも発言できる。ウェビナーでは発言権限を持つユーザーだけが発言できる。
// 発言権限を持つユーザーが記録されていない (この仕組みより前に作られた) ルームでは、
// 既に発言権限付きで参加しているかどうかで判断する。
func (r RoomWithParticipants) CanPublish(userID uuid.UUID) bool {
	if !r.IsWebinar {
		return true
	}
	if len(r.Publishers) > 0 {
		return slices.Contains(r.Publishers, userID)
	}
	return slices.ContainsFunc(r.Participants, func(p Participant) bool {
		return p.Name == userID.String() && p.CanPublish
	})
}

// RoomWithParticipants defines model for RoomWithParticipants.
type RoomWithParticipants struct {
	// IsWebinar ウェビナールームかどうか
	IsWebinar bool `json:"isWebinar"`

	// Metadata ルームに関連付けられたカスタム属性
	Metadata     *string       `json:"metadata,omitempty"`
	Participants []Participant `json:"participants"`

	// Publishers ウェビナーで発言権限 (= 管理権限) を持つユーザーのID
	Publishers []uuid.UUID `json:"-"`

	// RoomID ルームのID
	RoomID uuid.UUID `json:"roomId"`
}

// RoomStateManager はQallルーム状態を管理するインターフェース
type RoomStateManager interface {
	// InitializeRoomState LiveKit APIから現在のルーム状態を取得 (初期化時に利用)
	InitializeRoomState(ctx context.Context) error

	// AddParticipantToRoomState ルーム状態に参加者を追加
	AddParticipantToRoomState(room *livekit.Room, participant *livekit.ParticipantInfo)

	// UpdateParticipantCanPublish 参加者の発言権限を更新
	UpdateParticipantCanPublish(roomID string, participantID string, canPublish bool)

	// UpdateParticipant 参加者情報を更新
	UpdateParticipant(roomID string, participant *livekit.ParticipantInfo)

	// RemoveParticipant ルームから参加者を削除
	RemoveParticipant(roomID string, participantID string)

	// GetRoomsWithParticipantsByLiveKitServerAndSave LiveKitサーバーからルーム状態を取得して保存
	GetRoomsWithParticipantsByLiveKitServerAndSave(ctx context.Context) error

	// AddRoomState ルーム状態を追加
	AddRoomState(room RoomWithParticipants)

	// UpdateRoomMetadata ルームのメタデータ (ステータスと発言権限を持つユーザー) を更新
	UpdateRoomMetadata(roomID string, metadata Metadata)

	// RemoveRoomState ルーム状態を削除
	RemoveRoomState(roomID string)

	// GetRoomsByLiveKitServer LiveKitサーバーからルーム一覧を取得
	GetRoomsByLiveKitServer(ctx context.Context) (*livekit.ListRoomsResponse, error)

	// GetParticipantsByLiveKitServer LiveKitサーバーから参加者一覧を取得
	GetParticipantsByLiveKitServer(ctx context.Context, roomID string) (*livekit.ListParticipantsResponse, error)

	// GetRoomsWithParticipantsByLiveKitServer LiveKitサーバーからルーム状態と参加者一覧を取得
	GetRoomsWithParticipantsByLiveKitServer(ctx context.Context) ([]RoomWithParticipants, error)

	// GetState 現在のルーム状態を取得
	GetState() []RoomWithParticipants

	// GetRoomState 指定したルームの状態を取得
	GetRoomState(roomID string) *RoomWithParticipants
}
