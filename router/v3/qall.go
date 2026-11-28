package v3

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v5"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/qall"
	"github.com/twitchtv/twirp"
	"go.uber.org/zap"
)

// GetQallEndpoints GET /qall/endpoints
func (h *Handlers) GetQallEndpoints(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"endpoint": h.LiveKitHost,
	})
}

type soundboardItemResponse struct {
	SoundID   uuid.UUID  `json:"soundId"`
	SoundName string     `json:"soundName"`
	StampID   *uuid.UUID `json:"stampId"`
	CreatorID uuid.UUID  `json:"creatorId"`

	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func formatSoundboardItem(item *model.SoundboardItem) soundboardItemResponse {
	var stampID *uuid.UUID
	if item.StampID != nil && *item.StampID != uuid.Nil {
		id := *item.StampID
		stampID = &id
	}
	return soundboardItemResponse{
		SoundID:   item.ID,
		SoundName: item.Name,
		StampID:   stampID,
		CreatorID: item.CreatorID,
		ID:        item.ID,
		Name:      item.Name,
	}
}

// GetSoundboardItems GET /qall/soundboard
func (h *Handlers) GetSoundboardItems(c *echo.Context) error {
	items, err := h.Repo.GetAllSoundboardItems(c.Request().Context())
	if err != nil {
		return herror.InternalServerError(err)
	}
	res := make([]soundboardItemResponse, len(items))
	for i, item := range items {
		res[i] = formatSoundboardItem(item)
	}
	return c.JSON(http.StatusOK, res)
}

const soundboardItemNameMaxLength = 32

func firstFormFile(c *echo.Context, names ...string) (multipart.File, *multipart.FileHeader, error) {
	var lastErr error
	for _, name := range names {
		src, header, err := c.Request().FormFile(name)
		if err == nil {
			return src, header, nil
		}
		lastErr = err
	}
	return nil, nil, lastErr
}

func firstFormValue(c *echo.Context, names ...string) string {
	for _, name := range names {
		if v := strings.TrimSpace(c.FormValue(name)); v != "" {
			return v
		}
	}
	return ""
}

var soundboardExtensionTypes = map[string]string{
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".wave": "audio/wav",
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
}

func soundboardContentType(src multipart.File, header *multipart.FileHeader) (string, error) {
	if mimeType, _, err := mime.ParseMediaType(header.Header.Get(echo.HeaderContentType)); err == nil &&
		mimeType != "" && mimeType != echo.MIMEOctetStream {
		return mimeType, nil
	}
	if mimeType, ok := soundboardExtensionTypes[strings.ToLower(filepath.Ext(header.Filename))]; ok {
		return mimeType, nil
	}

	head := make([]byte, 512)
	n, err := io.ReadFull(src, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", err
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	switch detected := http.DetectContentType(head[:n]); detected {
	case "audio/wave":
		return "audio/wav", nil
	case "application/ogg":
		return "audio/ogg", nil
	default:
		return detected, nil
	}
}

// CreateSoundboardItem POST /qall/soundboard
func (h *Handlers) CreateSoundboardItem(c *echo.Context) error {
	src, uploadedFile, err := firstFormFile(c, "audio", "file")
	if err != nil {
		return herror.BadRequest("audio file is required")
	}
	defer src.Close()
	if uploadedFile.Size == 0 {
		return herror.BadRequest("non-empty file is required")
	}

	soundName := firstFormValue(c, "soundName", "name")
	if soundName == "" {
		return herror.BadRequest("soundName is required")
	}
	if utf8.RuneCountInString(soundName) > soundboardItemNameMaxLength {
		return herror.BadRequest(fmt.Sprintf("soundName must be at most %d characters", soundboardItemNameMaxLength))
	}

	var stampID *uuid.UUID
	if raw := firstFormValue(c, "stampId"); raw != "" {
		id, err := uuid.FromString(raw)
		if err != nil {
			return herror.BadRequest("invalid stampId")
		}
		if id != uuid.Nil {
			stampID = &id
		}
	}

	mimeType, err := soundboardContentType(src, uploadedFile)
	if err != nil {
		return herror.InternalServerError(err)
	}

	soundID := uuid.Must(uuid.NewV7())
	creatorID := getRequestUserID(c)
	if err := h.Soundboard.SaveSoundboardItem(c.Request().Context(), soundID, soundName, mimeType, model.FileTypeSoundboardItem, src, stampID, creatorID); err != nil {
		if errors.Is(err, qall.ErrInvalidSoundboardAudio) {
			return herror.BadRequest(err.Error())
		}
		return herror.InternalServerError(err)
	}

	return c.JSON(http.StatusOK, map[string]uuid.UUID{"soundId": soundID})
}

func (h *Handlers) isRoomParticipant(ctx context.Context, roomID uuid.UUID, userID uuid.UUID) (bool, error) {
	if roomState := h.QallRepo.GetRoomState(roomID.String()); roomState != nil {
		for _, participant := range roomState.Participants {
			if participant.Name == userID.String() {
				return true, nil
			}
		}
	}

	if h.LiveKitHost == "" {
		return false, nil
	}
	resp, err := h.QallRepo.GetParticipantsByLiveKitServer(ctx, roomID.String())
	if err != nil {
		var twErr twirp.Error
		if errors.As(err, &twErr) && twErr.Code() == twirp.NotFound {
			return false, nil
		}
		return false, err
	}
	for _, participant := range resp.Participants {
		if participant.Name == userID.String() {
			return true, nil
		}
	}
	return false, nil
}

// PlaySoundboardItem POST /qall/soundboard/play
func (h *Handlers) PlaySoundboardItem(c *echo.Context) error {
	var req struct {
		SoundID  string `json:"soundId"`
		RoomName string `json:"roomName"`
		RoomID   string `json:"roomId"`
	}
	if err := c.Bind(&req); err != nil {
		return herror.BadRequest(err)
	}

	soundID, err := uuid.FromString(req.SoundID)
	if err != nil {
		return herror.BadRequest("invalid sound ID")
	}

	rawRoomID := req.RoomName
	if rawRoomID == "" {
		rawRoomID = req.RoomID
	}
	roomID, err := uuid.FromString(rawRoomID)
	if err != nil {
		return herror.BadRequest("invalid room ID")
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	items, err := h.Repo.GetAllSoundboardItems(ctx)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if !slices.ContainsFunc(items, func(item *model.SoundboardItem) bool { return item.ID == soundID }) {
		return herror.NotFound("sound not found")
	}

	userID := getRequestUserID(c)
	isParticipant, err := h.isRoomParticipant(ctx, roomID, userID)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if !isParticipant {
		return herror.Forbidden("you are not a participant of the room")
	}
	if roomState := h.QallRepo.GetRoomState(roomID.String()); roomState != nil && !roomState.CanPublish(userID) {
		return herror.Forbidden("you are not allowed to speak in this webinar")
	}

	audioURL, err := h.Soundboard.GetURL(soundID)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if audioURL == "" {
		return herror.InternalServerError(errors.New("soundboard is not available with the current file storage"))
	}

	ingressClient := lksdk.NewIngressClient(h.LiveKitHost, h.LiveKitAPIKey, h.LiveKitAPISecret)

	ingReq := &livekit.CreateIngressRequest{
		InputType:           livekit.IngressInput_URL_INPUT,
		Name:                "soundboard-ingress",
		RoomName:            roomID.String(),
		ParticipantIdentity: "soundboard-" + uuid.Must(uuid.NewV7()).String(),
		ParticipantName:     "Soundboard " + soundID.String(),
		Url:                 audioURL,
	}

	info, err := ingressClient.CreateIngress(ctx, ingReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to create ingress: %v", err),
		})
	}

	resp := struct {
		IngressID string  `json:"ingressId"`
		URL       *string `json:"url"`
		StreamKey *string `json:"streamKey"`
	}{
		IngressID: info.IngressId,
		URL:       &info.Url,
		StreamKey: &info.StreamKey,
	}
	return c.JSON(http.StatusOK, resp)
}

// GetRoomState GET /qall/rooms
func (h *Handlers) GetRoomState(c *echo.Context) error {
	// Get room state from QallRepository
	roomState := h.QallRepo.GetState()

	return c.JSON(http.StatusOK, roomState)
}

// GetRoomMetadata GET /qall/rooms/:roomID/metadata
func (h *Handlers) GetRoomMetadata(c *echo.Context, roomID uuid.UUID) error {
	roomState := h.QallRepo.GetRoomState(roomID.String())
	if roomState == nil {
		return herror.NotFound("room not found")
	}
	return c.JSON(http.StatusOK, roomState.Metadata)
}

func fullParticipantPermission(canPublish bool) *livekit.ParticipantPermission {
	return &livekit.ParticipantPermission{
		CanSubscribe:      true,
		CanPublish:        canPublish,
		CanPublishData:    true,
		CanUpdateMetadata: true,
	}
}

func (h *Handlers) updateLiveKitRoomMetadata(ctx context.Context, roomID uuid.UUID, metadata qall.Metadata) error {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = lksdk.NewRoomServiceClient(h.LiveKitHost, h.LiveKitAPIKey, h.LiveKitAPISecret).
		UpdateRoomMetadata(ctx, &livekit.UpdateRoomMetadataRequest{
			Room:     roomID.String(),
			Metadata: string(raw),
		})
	return err
}

// PatchRoomMetadata PATCH /qall/rooms/:roomID/metadata
func (h *Handlers) PatchRoomMetadata(c *echo.Context) error {
	var req struct {
		Metadata string `json:"metadata"`
	}
	if err := c.Bind(&req); err != nil {
		return herror.BadRequest(err)
	}

	roomID, err := uuid.FromString(c.Param("roomID"))
	if err != nil {
		return herror.BadRequest("invalid room ID")
	}

	targetRoom := h.QallRepo.GetRoomState(roomID.String())
	if targetRoom == nil {
		return herror.NotFound("room not found")
	}

	// Verify user is a participant
	userID := getRequestUserID(c)
	if !slices.ContainsFunc(targetRoom.Participants, func(p qall.Participant) bool { return p.Name == userID.String() }) {
		return herror.Forbidden("you are not a participant of the room")
	}

	metadata := targetRoom.RoomMetadata()
	metadata.Status = req.Metadata
	if err := h.updateLiveKitRoomMetadata(c.Request().Context(), roomID, metadata); err != nil {
		return herror.InternalServerError(err)
	}

	h.QallRepo.UpdateRoomMetadata(roomID.String(), metadata)
	return c.NoContent(http.StatusOK)
}

type roomParticipantUpdate struct {
	UserID     string `json:"userId"`
	CanPublish bool   `json:"canPublish"`
}

type roomParticipantsUpdateRequest struct {
	Users []roomParticipantUpdate `json:"users"`
}

func parseRoomParticipantsUpdate(body []byte) ([]roomParticipantUpdate, error) {
	var list []roomParticipantsUpdateRequest
	if err := json.Unmarshal(body, &list); err == nil {
		var users []roomParticipantUpdate
		for _, req := range list {
			users = append(users, req.Users...)
		}
		return users, nil
	}
	var req roomParticipantsUpdateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return req.Users, nil
}

type roomParticipantResult struct {
	ParticipantID string `json:"participantId"`
	Status        string `json:"status"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

// PatchRoomParticipants PATCH /qall/rooms/:roomID/participants
func (h *Handlers) PatchRoomParticipants(c *echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return herror.BadRequest(err)
	}
	updates, err := parseRoomParticipantsUpdate(body)
	if err != nil {
		return herror.BadRequest("invalid request body")
	}
	if len(updates) == 0 {
		return herror.BadRequest("users are required")
	}

	roomID, err := uuid.FromString(c.Param("roomID"))
	if err != nil {
		return herror.BadRequest("invalid room ID")
	}

	roomState := h.QallRepo.GetRoomState(roomID.String())
	if roomState == nil {
		return herror.NotFound("room not found")
	}
	if !roomState.IsWebinar {
		return herror.BadRequest("speaking permissions can only be changed in a webinar")
	}

	userID := getRequestUserID(c)
	if !roomState.CanPublish(userID) {
		return herror.Forbidden("you are not allowed to update participants")
	}

	publishers := slices.Clone(roomState.Publishers)
	if len(publishers) == 0 {
		for _, p := range roomState.Participants {
			if id, err := uuid.FromString(p.Name); err == nil && p.CanPublish && !slices.Contains(publishers, id) {
				publishers = append(publishers, id)
			}
		}
	}
	targets := make([]uuid.UUID, 0, len(updates))
	for _, u := range updates {
		target, err := uuid.FromString(u.UserID)
		if err != nil {
			return herror.BadRequest("invalid user ID")
		}
		targets = append(targets, target)
		if u.CanPublish {
			if !slices.Contains(publishers, target) {
				publishers = append(publishers, target)
			}
		} else {
			publishers = slices.DeleteFunc(publishers, func(id uuid.UUID) bool { return id == target })
		}
	}
	if len(publishers) == 0 {
		return herror.BadRequest("at least one user must keep the speaking permission")
	}

	ctx := c.Request().Context()
	metadata := roomState.RoomMetadata()
	metadata.Publishers = publishers
	if err := h.updateLiveKitRoomMetadata(ctx, roomID, metadata); err != nil {
		return herror.InternalServerError(err)
	}
	h.QallRepo.UpdateRoomMetadata(roomID.String(), metadata)

	livekitClient := lksdk.NewRoomServiceClient(h.LiveKitHost, h.LiveKitAPIKey, h.LiveKitAPISecret)
	results := make([]roomParticipantResult, 0, len(targets))
	for _, target := range targets {
		canPublish := metadata.IsPublisher(target)
		result := roomParticipantResult{ParticipantID: target.String(), Status: "success"}
		for _, p := range roomState.Participants {
			if p.Name != target.String() {
				continue
			}
			_, err := livekitClient.UpdateParticipant(ctx, &livekit.UpdateParticipantRequest{
				Room:       roomID.String(),
				Identity:   p.Identity,
				Permission: fullParticipantPermission(canPublish),
			})
			if err != nil {
				result.Status = "error"
				result.ErrorMessage = err.Error()
				continue
			}
			h.QallRepo.UpdateParticipantCanPublish(roomID.String(), p.Identity, canPublish)
		}
		results = append(results, result)
	}

	return c.JSON(http.StatusOK, map[string]any{"results": results})
}

// GetLiveKitToken GET /qall/token
func (h *Handlers) GetLiveKitToken(c *echo.Context) error {
	ctx := c.Request().Context()
	// 1) roomクエリパラメータ取得 (必須)
	room := c.QueryParam("roomId")
	if room == "" {
		return herror.BadRequest("room query parameter is required")
	}

	roomID, err := uuid.FromString(room)
	if err != nil {
		return herror.BadRequest("invalid room Id")
	}

	if !h.ChannelManager.PublicChannelTree(ctx).IsChannelPresent(roomID) {
		return herror.NotFound("channel not found")
	}

	isWebinar := c.QueryParam("isWebinar") == "true"
	userID := getRequestUserID(c)

	roomState := h.QallRepo.GetRoomState(roomID.String())
	if roomState == nil {
		metadata := qall.Metadata{IsWebinar: isWebinar}
		if isWebinar {
			metadata.Publishers = []uuid.UUID{userID}
		}
		metadataStr, err := json.Marshal(metadata)
		if err != nil {
			return herror.InternalServerError(err)
		}
		lkclient := lksdk.NewRoomServiceClient(h.LiveKitHost, h.LiveKitAPIKey, h.LiveKitAPISecret)
		created, err := lkclient.CreateRoom(ctx, &livekit.CreateRoomRequest{
			Name:     room,
			Metadata: string(metadataStr),
		})
		if err != nil {
			return herror.InternalServerError(err)
		}
		h.QallRepo.AddRoomState(qall.RoomStateFromLiveKit(roomID, created.Metadata))
		roomState = h.QallRepo.GetRoomState(roomID.String())
		if roomState == nil {
			return herror.InternalServerError(errors.New("failed to create the room state"))
		}
	}

	at := auth.NewAccessToken(h.LiveKitAPIKey, h.LiveKitAPISecret)
	grant := &auth.VideoGrant{
		RoomJoin:             true,
		Room:                 room,
		CanPublish:           new(roomState.CanPublish(userID)),
		CanPublishData:       new(true),
		CanUpdateOwnMetadata: new(true),
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

	return c.JSON(http.StatusOK, map[string]string{"token": livekitToken})
}

// LiveKitWebhook POST /qall/webhook
func (h *Handlers) LiveKitWebhook(c *echo.Context) error {
	// Authプロバイダーを初期化
	authProvider := auth.NewSimpleKeyProvider(h.LiveKitAPIKey, h.LiveKitAPISecret)

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
