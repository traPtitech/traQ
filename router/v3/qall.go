package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
	lksdk "github.com/livekit/server-sdk-go"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/qall"
	"go.uber.org/zap"
)

// GetSoundboardItems
func (h *Handlers) GetSoundboardItems(c echo.Context) error {
	items, err := h.Repo.GetAllSoundboardItems()
	if err != nil {
		return herror.InternalServerError(err)
	}
	return c.JSON(http.StatusOK, items)
}

// CreateSoundboardItem
func (h *Handlers) CreateSoundboardItem(c echo.Context) error {
	src, uploadedFile, err := c.Request().FormFile("file")
	if err != nil {
		return herror.BadRequest(err)
	}
	defer src.Close()
	if uploadedFile.Size == 0 {
		return herror.BadRequest("non-empty file is required")
	}

	mimeType := uploadedFile.Header.Get(echo.HeaderContentType)
	soundName := c.FormValue("name")
	creatorID := uuid.FromStringOrNil(c.FormValue("creatorId"))
	stampID := uuid.FromStringOrNil(c.FormValue("stampId"))

	if err := h.Soundboard.SaveSoundboardItem(uuid.Must(uuid.NewV7()), soundName, mimeType, model.FileTypeSoundboardItem, src, &stampID, creatorID); err != nil {
		return herror.InternalServerError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// GetRoomState returns the current state of all rooms
func (h *Handlers) GetRoomState(c echo.Context) error {
	// Get room state from QallRepository
	roomState := h.QallRepo.GetState()

	return c.JSON(http.StatusOK, roomState)
}

// GetRoomMetadata returns the metadata of the specified room
func (h *Handlers) GetRoomMetadata(c echo.Context, roomID uuid.UUID) error {
	roomState := h.QallRepo.GetState()
	for _, state := range roomState {
		if state.RoomId == roomID {
			return c.JSON(http.StatusOK, state.Metadata)
		}
	}
	return herror.NotFound("room not found")
}

// PatchRoomMetadata updates the metadata of the specified room
func (h *Handlers) PatchRoomMetadata(c echo.Context) error {
	type roomMetadataRequest struct {
		Metadata string `json:"metadata"`
	}
	var req roomMetadataRequest
	if err := c.Bind(&req); err != nil {
		return herror.BadRequest(err)
	}

	roomID, err := uuid.FromString(c.Param("roomID"))
	if err != nil {
		return herror.BadRequest("invalid room ID")
	}

	userID := getRequestUserID(c)

	livekitClient := lksdk.NewRoomServiceClient(h.Config.LiveKitHost, h.Config.LiveKitApiKey, h.Config.LiveKitApiSecret)
	roomState := h.QallRepo.GetState()
	for _, state := range roomState {
		if state.RoomId == roomID {
			// Check if the user is one of the participants
			isParticipant := false
			for _, participant := range state.Participants {
				if *participant.Name == userID.String() {
					isParticipant = true
					break
				}
			}
			if !isParticipant {
				return herror.Forbidden("you are not a participant of the room")
			}

			// Update the metadata
			metadata := qall.Metadata{
				Status:    req.Metadata,
				IsWebinar: *state.IsWebinar,
			}
			_, err := livekitClient.UpdateRoomMetadata(c.Request().Context(), &livekit.UpdateRoomMetadataRequest{
				Room:     state.RoomId.String(),
				Metadata: req.Metadata,
			})
			if err != nil {
				return herror.InternalServerError(err)
			}

			h.QallRepo.UpdateRoomMetadata(roomID.String(), metadata)
			return c.NoContent(http.StatusNoContent)
		}
	}
	return herror.NotFound("room not found")
}

// PatchRoomParticipants updates the participants of the specified room
func (h *Handlers) PatchRoomParticipants(c echo.Context) error {
	type RoomParticipantUpdate struct {
		UserID     string `json:"userId"`
		CanPublish bool   `json:"canPublish"`
	}

	type RoomParticipantsUpdateRequest struct {
		Users []RoomParticipantUpdate `json:"users"`
	}

	var req RoomParticipantsUpdateRequest
	var succeedUsers []string
	var failedUsers = map[string]string{}
	if err := c.Bind(&req); err != nil {
		return herror.BadRequest(err)
	}

	userID := getRequestUserID(c)
	roomID, err := uuid.FromString(c.Param("roomID"))
	if err != nil {
		return herror.BadRequest("invalid room ID")
	}

	// ルームが存在するか確認
	for _, roomState := range h.QallRepo.GetState() {
		if roomState.RoomId == roomID {
			// userがcanPublishかどうかを確認
			canPublish := false
			for _, participant := range roomState.Participants {
				if *participant.Name == userID.String() {
					canPublish = *participant.CanPublish
					break
				}
			}
			if !canPublish {
				return herror.Forbidden("you are not allowed to update participants")
			}
			livekitClient := lksdk.NewRoomServiceClient(h.Config.LiveKitHost, h.Config.LiveKitApiKey, h.Config.LiveKitApiSecret)
			for _, participant := range req.Users {
				for _, roomParticipant := range roomState.Participants {
					if *roomParticipant.Name == participant.UserID {
						_, err := livekitClient.UpdateParticipant(c.Request().Context(), &livekit.UpdateParticipantRequest{
							Room:     roomID.String(),
							Identity: *roomParticipant.Identity,
							Permission: &livekit.ParticipantPermission{
								CanPublish: participant.CanPublish,
							},
						})
						if err != nil {
							failedUsers[participant.UserID] = err.Error()
						} else {
							succeedUsers = append(succeedUsers, participant.UserID)
							h.QallRepo.UpdateParticipantCanPublish(roomID.String(), *roomParticipant.Identity, participant.CanPublish)
						}
					}
				}
			}
			break
		}
	}

	response := make([]map[string]string, 0)

	// Add successful participants
	for _, id := range succeedUsers {
		response = append(response, map[string]string{
			"participantId": id,
			"status":        "success",
		})
	}

	// Add failed participants
	for _, id := range failedUsers {
		response = append(response, map[string]string{
			"participantId": id,
			"status":        "failed",
			"error":         failedUsers[id],
		})
	}

	return c.JSON(http.StatusOK, response)

}

// GetLiveKitToken returns a token for LiveKit
func (h *Handlers) GetLiveKitToken(c echo.Context) error {
	// 1) roomクエリパラメータ取得 (必須)
	room := c.QueryParam("room")
	if room == "" {
		return herror.BadRequest("room query parameter is required")
	}

	roomID, err := uuid.FromString(room)
	if err != nil {
		return herror.BadRequest("invalid room ID")
	}

	if !h.ChannelManager.PublicChannelTree().IsChannelPresent(roomID) {
		return herror.NotFound("channel not found")
	}

	isWebinar := c.QueryParam("isWebinar") == "true"

	userID := getRequestUserID(c)

	// 6-2) ルームが存在するか確認
	isExistingRoom := false
	for _, roomState := range h.QallRepo.GetState() {
		if roomState.RoomId == roomID {
			isExistingRoom = true
			break
		}
	}

	if isExistingRoom {
		// ルームが存在して、webinar=true の場合はCanPublish=false
		for _, roomState := range h.QallRepo.GetState() {
			if roomState.RoomId.String() == room {
				if roomState.IsWebinar != nil && *roomState.IsWebinar {
					isWebinar = true
				}
				break
			}
		}
	}

	// 7) VideoGrant にルーム名、CanPublishData=true を設定
	// ルームが存在しない場合はCanPublish=true
	// ルームが存在して、webinar=true の場合はCanPublish=false
	// ただし、自分がすでに参加していてCanPublish=true の場合はCanPublish=true
	isAlreadyCanPublish := false
	for _, roomState := range h.QallRepo.GetState() {
		if roomState.RoomId == roomID {
			for _, participant := range roomState.Participants {
				if *participant.Name == userID.String() {
					isAlreadyCanPublish = *participant.CanPublish
					break
				}
			}
			break
		}
	}
	at := auth.NewAccessToken(h.Config.LiveKitApiKey, h.Config.LiveKitApiSecret)
	grant := &auth.VideoGrant{
		RoomJoin:             true,
		Room:                 room,
		CanPublish:           func(b bool) *bool { return &b }(!(isWebinar && isExistingRoom) || isAlreadyCanPublish),
		CanPublishData:       func(b bool) *bool { return &b }(true),
		CanUpdateOwnMetadata: func(b bool) *bool { return &b }(true),
	}
	randomUUID, err := uuid.NewV7()
	if err != nil {
		return herror.InternalServerError(err)
	}
	userIdentity := fmt.Sprintf("%s_%s", userID, randomUUID.String())
	at.SetVideoGrant(grant).
		SetIdentity(userIdentity).
		SetName(userID.String()).
		SetValidFor(24 * time.Hour)

	livekitToken, err := at.ToJWT()
	if err != nil {
		return herror.InternalServerError(err)
	}

	if !isExistingRoom {
		metadata := qall.Metadata{
			Status:    "",
			IsWebinar: isWebinar,
		}
		metadataStr, err := json.Marshal(metadata)
		if err != nil {
			return herror.InternalServerError(err)
		}
		lkclient := lksdk.NewRoomServiceClient(h.Config.LiveKitHost, h.Config.LiveKitApiKey, h.Config.LiveKitApiSecret)
		_, err = lkclient.CreateRoom(c.Request().Context(), &livekit.CreateRoomRequest{
			Name:     room,
			Metadata: string(metadataStr),
		})
		if err != nil {
			return herror.InternalServerError(err)
		}
		// ルームが存在しない場合は新規作成
		emptyMetadata := ""
		roomWithParticipants := qall.RoomWithParticipants{
			IsWebinar:    &isWebinar,
			Metadata:     &emptyMetadata,
			RoomId:       roomID,
			Participants: []qall.Participant{},
		}
		h.QallRepo.AddRoomState(roomWithParticipants)
	}

	return c.JSON(http.StatusOK, map[string]string{"Token": livekitToken})

}

// LiveKitWebhook handles webhooks from LiveKit server
func (h *Handlers) LiveKitWebhook(c echo.Context) error {
	// Authプロバイダーを初期化
	authProvider := auth.NewSimpleKeyProvider(h.Config.LiveKitApiKey, h.Config.LiveKitApiSecret)

	// Webhookイベントを受け取る
	event, err := webhook.ReceiveWebhookEvent(c.Request(), authProvider)
	if err != nil {
		h.Logger.Error("failed to validate webhook", zap.Error(err))
		return herror.BadRequest("Failed to validate webhook")
	}

	// ルーム状態を更新
	switch event.Event {
	case webhook.EventRoomFinished:
		h.Logger.Info("Room finished", zap.String("room", event.Room.Name))
		// ルーム状態を削除
		h.QallRepo.RemoveRoomState(event.Room.Name)

	case webhook.EventParticipantJoined:
		h.Logger.Info("Participant joined",
			zap.String("room", event.Room.Name),
			zap.String("participant", event.Participant.Identity),
			zap.String("name", event.Participant.Name))
		// 参加者を追加
		h.QallRepo.AddParticipantToRoomState(event.Room, event.Participant)

	case webhook.EventParticipantLeft:
		h.Logger.Info("Participant left",
			zap.String("room", event.Room.Name),
			zap.String("participant", event.Participant.Identity),
			zap.String("name", event.Participant.Name))
		// 参加者を削除
		h.QallRepo.RemoveParticipant(event.Room.Name, event.Participant.Identity)

	default:
		h.Logger.Info("Unhandled webhook event", zap.String("event", string(event.Event)))
	}

	return c.NoContent(http.StatusOK)
}
