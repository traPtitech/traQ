package v3

import (
	"errors"
	"net/http"
	"time"

	vd "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v5"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/extension/herror"
	"github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/service/scheduled"
)

type PostScheduledMessageRequest struct {
	ChannelID    uuid.UUID   `json:"channelId"`
	Content      string      `json:"content"`
	DraftContent string      `json:"draftContent"`
	ScheduledAt  time.Time   `json:"scheduledAt"`
	FileIDs      []uuid.UUID `json:"fileIds"`
}

func (r PostScheduledMessageRequest) Validate() error {
	if r.ChannelID == uuid.Nil {
		return errors.New("channelId is required")
	}
	if !r.ScheduledAt.After(time.Now()) {
		return errors.New("scheduledAt must be in the future")
	}
	if len(r.FileIDs) > 100 {
		return errors.New("too many attachments")
	}
	seen := make(map[uuid.UUID]bool, len(r.FileIDs))
	for _, id := range r.FileIDs {
		if id == uuid.Nil || seen[id] {
			return errors.New("invalid or duplicate fileId")
		}
		seen[id] = true
	}
	return vd.ValidateStruct(&r,
		vd.Field(&r.Content, vd.Required, vd.RuneLength(1, 10000)),
		vd.Field(&r.DraftContent, vd.RuneLength(0, 10000)),
	)
}

type scheduledMessageResponse struct {
	ID           uuid.UUID   `json:"id"`
	ChannelID    uuid.UUID   `json:"channelId"`
	Content      string      `json:"content"`
	DraftContent string      `json:"draftContent"`
	ScheduledAt  time.Time   `json:"scheduledAt"`
	CreatedAt    time.Time   `json:"createdAt"`
	FileIDs      []uuid.UUID `json:"fileIds"`
	Status       string      `json:"status"`
	Failure      string      `json:"failure"`
}

func formatScheduledMessage(m *model.ScheduledMessage) scheduledMessageResponse {
	return scheduledMessageResponse{
		ID: m.ID, ChannelID: m.ChannelID, Content: m.Content, DraftContent: m.DraftContent,
		ScheduledAt: m.ScheduledAt, CreatedAt: m.CreatedAt, FileIDs: m.FileIDs(), Status: m.Status, Failure: m.Failure,
	}
}

func (h *Handlers) GetMyScheduledMessages(c *echo.Context) error {
	list, err := h.Repo.GetScheduledMessages(c.Request().Context(), getRequestUserID(c))
	if err != nil {
		return herror.InternalServerError(err)
	}
	result := make([]scheduledMessageResponse, len(list))
	for i, m := range list {
		result[i] = formatScheduledMessage(m)
	}
	return c.JSON(http.StatusOK, result)
}

func (h *Handlers) PostScheduledMessage(c *echo.Context) error {
	var req PostScheduledMessageRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	m := &model.ScheduledMessage{
		ID: uuid.Must(uuid.NewV7()), UserID: getRequestUserID(c), ChannelID: req.ChannelID,
		Content: req.Content, DraftContent: req.DraftContent, ScheduledAt: req.ScheduledAt,
	}
	for _, id := range req.FileIDs {
		m.Attachments = append(m.Attachments, model.ScheduledMessageFile{FileID: id})
	}
	w := scheduled.Worker{Repo: h.Repo, Channels: h.ChannelManager, RBAC: h.RBAC}
	_, failure, err := w.Validate(c.Request().Context(), m)
	if err != nil {
		return herror.InternalServerError(err)
	}
	if failure != "" {
		return herror.Forbidden(failure)
	}
	if err := h.Repo.CreateScheduledMessage(c.Request().Context(), m); err != nil {
		switch {
		case errors.Is(err, repository.ErrForbidden):
			return herror.BadRequest("attachments must be your private scheduled-message uploads")
		case errors.Is(err, repository.ErrAlreadyExists):
			return herror.BadRequest("attachment is already reserved")
		case repository.IsArgError(err):
			return herror.BadRequest(err)
		default:
			return herror.InternalServerError(err)
		}
	}
	return c.JSON(http.StatusCreated, formatScheduledMessage(m))
}

func (h *Handlers) CancelScheduledMessage(c *echo.Context) error {
	id, err := uuid.FromString(c.Param("scheduledMessageID"))
	if err != nil {
		return herror.BadRequest("invalid scheduledMessageID")
	}
	m, err := h.Repo.CancelScheduledMessage(c.Request().Context(), id, getRequestUserID(c))
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return herror.NotFound()
		case errors.Is(err, repository.ErrAlreadyExists):
			return c.JSON(http.StatusConflict, map[string]string{"message": "message has already been sent"})
		default:
			return herror.InternalServerError(err)
		}
	}
	for _, fid := range m.FileIDs() {
		if err := h.FileManager.Delete(c.Request().Context(), fid); err != nil && !errors.Is(err, file.ErrNotFound) {
			h.Logger.Warn("failed to remove canceled attachment")
		}
	}
	return c.NoContent(http.StatusNoContent)
}
