package qall

import (
	"context"
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

	// RoomID ルームのID
	RoomID uuid.UUID `json:"roomId"`
}

// RoomStateManager はQallルーム状態を管理するインターフェース
type RoomStateManager interface {
	// InitializeRoomState LiveKit APIから現在のルーム状態を取得 (初期化時に利用)
	InitializeRoomState() error

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

	// UpdateRoomMetadata ルームのメタデータを更新
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
