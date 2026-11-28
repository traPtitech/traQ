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

// soundboardItemResponse サウンドボードアイテムのレスポンス
//
// API 定義のフィールド (soundId / soundName) に加えて、
// 以前の実装が返していた id / name も後方互換のために返す
type soundboardItemResponse struct {
	SoundID   uuid.UUID  `json:"soundId"`
	SoundName string     `json:"soundName"`
	StampID   *uuid.UUID `json:"stampId"`
	CreatorID uuid.UUID  `json:"creatorId"`

	// Deprecated: soundId を使う
	ID uuid.UUID `json:"id"`
	// Deprecated: soundName を使う
	Name string `json:"name"`
}

func formatSoundboardItem(item *model.SoundboardItem) soundboardItemResponse {
	var stampID *uuid.UUID
	// 以前はスタンプ未指定でも uuid.Nil が保存されていたので、それも未指定として扱う
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

// soundboardItemNameMaxLength サウンド名の最大文字数 (DB のカラム長)
const soundboardItemNameMaxLength = 32

// firstFormFile 指定した名前のうち、最初に見つかったフォームのファイルを返す
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

// firstFormValue 指定した名前のうち、最初に空でない値を返す
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

// soundboardContentType アップロードされた音声ファイルの MIME タイプを決める
//
// クライアントによっては application/octet-stream で送ってくるので、
// その場合はファイル名の拡張子、次にファイルの中身から推定する
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
//
// API 定義のフィールド (audio / soundName / stampId) に加えて、
// 以前の実装が受け付けていた file / name も後方互換のために受け付ける。
// 作成者は常にリクエストしたユーザーになる (以前の creatorId フィールドは無視する)。
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

// isRoomParticipant ユーザーがルームに参加しているかどうか
//
// traQ が持つルーム状態で見つからなければ、取りこぼし (Webhook の欠落など) に備えて LiveKit に問い合わせる
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
		// ルームが存在しない場合は参加していないものとして扱う
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
//
// API 定義のフィールド (roomName) に加えて、以前の実装が受け付けていた roomId も後方互換のために受け付ける
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

	// サウンドが存在するか確認
	items, err := h.Repo.GetAllSoundboardItems(ctx)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if !slices.ContainsFunc(items, func(item *model.SoundboardItem) bool { return item.ID == soundID }) {
		return herror.NotFound("sound not found")
	}

	// ルームに参加しているか確認
	userID := getRequestUserID(c)
	isParticipant, err := h.isRoomParticipant(ctx, roomID, userID)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if !isParticipant {
		return herror.Forbidden("you are not a participant of the room")
	}

	// S3ファイルキー = soundId として署名付きURL生成
	audioURL, err := h.Soundboard.GetURL(soundID)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if audioURL == "" {
		return herror.InternalServerError(errors.New("soundboard is not available with the current file storage"))
	}

	ingressClient := lksdk.NewIngressClient(h.LiveKitHost, h.LiveKitAPIKey, h.LiveKitAPISecret)

	// 同じ音を同時に鳴らしても前の再生が切断されないように、再生ごとに別の Identity を使う
	// ("soundboard-" で始まることはクライアントがサウンドボードの参加者を見分けるのに使う)
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

	userID := getRequestUserID(c)

	livekitClient := lksdk.NewRoomServiceClient(h.LiveKitHost, h.LiveKitAPIKey, h.LiveKitAPISecret)
	// Find the room
	targetRoom := h.QallRepo.GetRoomState(roomID.String())

	if targetRoom == nil {
		return herror.NotFound("room not found")
	}

	// Verify user is a participant
	isParticipant := false
	for _, participant := range targetRoom.Participants {
		if participant.Name == userID.String() {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		return herror.Forbidden("you are not a participant of the room")
	}

	// Update metadata
	metadata := qall.Metadata{
		Status:    req.Metadata,
		IsWebinar: targetRoom.IsWebinar,
	}

	_, err = livekitClient.UpdateRoomMetadata(c.Request().Context(), &livekit.UpdateRoomMetadataRequest{
		Room:     roomID.String(),
		Metadata: req.Metadata,
	})
	if err != nil {
		return herror.InternalServerError(err)
	}

	h.QallRepo.UpdateRoomMetadata(roomID.String(), metadata)
	return herror.NotFound("room not found")
}

// PatchRoomParticipants DELETE /qall/rooms/:roomID/participants
func (h *Handlers) PatchRoomParticipants(c *echo.Context) error {
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
	roomState := h.QallRepo.GetRoomState(roomID.String())
	if roomState == nil {
		return herror.NotFound("room not found")
	}

	// userがcanPublishかどうかを確認
	canPublish := false
	for _, participant := range roomState.Participants {
		if participant.Name == userID.String() {
			canPublish = participant.CanPublish
			break
		}
	}
	if !canPublish {
		return herror.Forbidden("you are not allowed to update participants")
	}

	// Update participants
	livekitClient := lksdk.NewRoomServiceClient(h.LiveKitHost, h.LiveKitAPIKey, h.LiveKitAPISecret)
	for _, participant := range req.Users {
		for _, roomParticipant := range roomState.Participants {
			if roomParticipant.Name == participant.UserID {
				_, err := livekitClient.UpdateParticipant(c.Request().Context(), &livekit.UpdateParticipantRequest{
					Room:     roomID.String(),
					Identity: roomParticipant.Identity,
					Permission: &livekit.ParticipantPermission{
						CanPublish: participant.CanPublish,
					},
				})
				if err != nil {
					failedUsers[participant.UserID] = err.Error()
				} else {
					succeedUsers = append(succeedUsers, participant.UserID)
					h.QallRepo.UpdateParticipantCanPublish(roomID.String(), roomParticipant.Identity, participant.CanPublish)
				}
			}
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

	// 6-2) ルームが存在するか確認
	roomState := h.QallRepo.GetRoomState(roomID.String())

	// ルームが存在して、webinar=true の場合はCanPublish=false
	isExistingRoom := roomState != nil
	if isExistingRoom && roomState.IsWebinar {
		isWebinar = true
	}

	// 7) VideoGrant にルーム名、CanPublishData=true を設定
	// ルームが存在しない場合はCanPublish=true
	// ルームが存在して、webinar=true の場合はCanPublish=false
	// ただし、自分がすでに参加していてCanPublish=true の場合はCanPublish=true
	isAlreadyCanPublish := false
	if roomState != nil {
		for _, participant := range roomState.Participants {
			if participant.Name == userID.String() {
				isAlreadyCanPublish = participant.CanPublish
				break
			}
		}
	}

	at := auth.NewAccessToken(h.LiveKitAPIKey, h.LiveKitAPISecret)
	grant := &auth.VideoGrant{
		RoomJoin:             true,
		Room:                 room,
		CanPublish:           func(b bool) *bool { return &b }((!isWebinar || !isExistingRoom) || isAlreadyCanPublish),
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
		lkclient := lksdk.NewRoomServiceClient(h.LiveKitHost, h.LiveKitAPIKey, h.LiveKitAPISecret)
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
			IsWebinar:    isWebinar,
			Metadata:     &emptyMetadata,
			RoomID:       roomID,
			Participants: []qall.Participant{},
		}
		h.QallRepo.AddRoomState(roomWithParticipants)
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
