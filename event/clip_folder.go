package event

import (
	"github.com/gofrs/uuid"

	"github.com/traPtitech/traQ/model"
)

// ClipFolderCreatedPayload ClipFolderCreated のペイロード
type ClipFolderCreatedPayload struct {
	UserID       uuid.UUID         `hub:"user_id"`
	ClipFolderID uuid.UUID         `hub:"clip_folder_id"`
	ClipFolder   *model.ClipFolder `hub:"clip_folder"`
}

func (ClipFolderCreatedPayload) TopicName() string { return ClipFolderCreated }

// ClipFolderUpdatedPayload ClipFolderUpdated のペイロード
type ClipFolderUpdatedPayload struct {
	UserID        uuid.UUID         `hub:"user_id"`
	ClipFolderID  uuid.UUID         `hub:"clip_folder_id"`
	OldClipFolder *model.ClipFolder `hub:"old_clip_folder"`
	ClipFolder    *model.ClipFolder `hub:"clip_folder"`
}

func (ClipFolderUpdatedPayload) TopicName() string { return ClipFolderUpdated }

// ClipFolderDeletedPayload ClipFolderDeleted のペイロード
type ClipFolderDeletedPayload struct {
	UserID       uuid.UUID         `hub:"user_id"`
	ClipFolderID uuid.UUID         `hub:"clip_folder_id"`
	ClipFolder   *model.ClipFolder `hub:"clip_folder"`
}

func (ClipFolderDeletedPayload) TopicName() string { return ClipFolderDeleted }

// ClipFolderMessageDeletedPayload ClipFolderMessageDeleted のペイロード
type ClipFolderMessageDeletedPayload struct {
	UserID              uuid.UUID                `hub:"user_id"`
	ClipFolderID        uuid.UUID                `hub:"clip_folder_id"`
	ClipFolderMessageID uuid.UUID                `hub:"clip_folder_message_id"`
	ClipFolderMessage   *model.ClipFolderMessage `hub:"clip_folder_message"`
}

func (ClipFolderMessageDeletedPayload) TopicName() string { return ClipFolderMessageDeleted }

// ClipFolderMessageAddedPayload ClipFolderMessageAdded のペイロード
type ClipFolderMessageAddedPayload struct {
	UserID              uuid.UUID                `hub:"user_id"`
	ClipFolderID        uuid.UUID                `hub:"clip_folder_id"`
	ClipFolderMessageID uuid.UUID                `hub:"clip_folder_message_id"`
	ClipFolderMessage   *model.ClipFolderMessage `hub:"clip_folder_message"`
}

func (ClipFolderMessageAddedPayload) TopicName() string { return ClipFolderMessageAdded }

var (
	_ Payload = ClipFolderCreatedPayload{}
	_ Payload = ClipFolderUpdatedPayload{}
	_ Payload = ClipFolderDeletedPayload{}
	_ Payload = ClipFolderMessageDeletedPayload{}
	_ Payload = ClipFolderMessageAddedPayload{}
)
