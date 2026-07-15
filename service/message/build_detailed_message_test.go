// atode fairu kuttsukeru

package message

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
)

func fileEmbed(id uuid.UUID) string {
	return fmt.Sprintf(`!{"raw":"file","type":"file","id":"%s"}`, id.String())
}

func citationEmbed(id uuid.UUID) string {
	return fmt.Sprintf(`!{"raw":"message","type":"message","id":"%s"}`, id.String())
}

func TestManager_buildDetailedMessage(t *testing.T) {
	t.Parallel()

	t.Run("attachments with quotes both disabled", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mgr, _, _, _ := setupM(ctrl)
		m := mgr.(*manager)

		fileID := uuid.NewV3(uuid.Nil, "f1")
		mm := &model.Message{
			ID:   uuid.NewV3(uuid.Nil, "m1"),
			Text: fileEmbed(fileID),
		}

		result, _ := m.buildDetailedMessage(context.TODO(), mm, false, false, uuid.NewV3(uuid.Nil, "u1"))
		assert.Equal(t, mm.ID, result.ID)
		assert.Nil(t, result.Attachments)
		assert.Nil(t, result.Quotes)
	})

	t.Run("attachments only", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mgr, _, repo, _ := setupM(ctrl)
		m := mgr.(*manager)

		fileID := uuid.NewV3(uuid.Nil, "f1")
		userID := uuid.NewV3(uuid.Nil, "u1")
		mm := &model.Message{ID: uuid.NewV3(uuid.Nil, "m1"), Text: fileEmbed(fileID)}

		repo.MockFileRepository.EXPECT().IsFileAccessible(gomock.Any(), fileID, userID).Return(true, nil).AnyTimes()
		repo.MockFileRepository.EXPECT().GetFileMeta(gomock.Any(), fileID).Return(&model.FileMeta{ID: fileID}, nil).Times(1)

		result, _ := m.buildDetailedMessage(context.TODO(), mm, true, false, uuid.NewV3(uuid.Nil, "u1"))
		if assert.Len(t, result.Attachments, 1) {
			assert.Equal(t, fileID, result.Attachments[0].ID)
		}
		assert.Nil(t, result.Quotes)
	})

	t.Run("GetFileMeta error stops further attachment resolution", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mgr, _, repo, _ := setupM(ctrl)
		m := mgr.(*manager)

		fileID1 := uuid.NewV3(uuid.Nil, "f1")
		fileID2 := uuid.NewV3(uuid.Nil, "f2")
		userID := uuid.NewV3(uuid.Nil, "u1")
		mm := &model.Message{
			ID:   uuid.NewV3(uuid.Nil, "m1"),
			Text: fileEmbed(fileID1) + fileEmbed(fileID2),
		}

		repo.MockFileRepository.EXPECT().IsFileAccessible(gomock.Any(), fileID1, userID).Return(true, nil).AnyTimes()
		repo.MockFileRepository.EXPECT().GetFileMeta(gomock.Any(), fileID1).Return(nil, repository.ErrNotFound).Times(1)

		result, _ := m.buildDetailedMessage(context.TODO(), mm, true, false, uuid.NewV3(uuid.Nil, "u1"))
		assert.Empty(t, result.Attachments)
	})

	t.Run("quotes only", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mgr, _, repo, _ := setupM(ctrl)
		m := mgr.(*manager)

		quoteID := uuid.NewV3(uuid.Nil, "q1")
		mm := &model.Message{ID: uuid.NewV3(uuid.Nil, "m1"), Text: citationEmbed(quoteID)}

		quoted := &model.Message{ID: quoteID, Text: "Quoted Text"}
		repo.MockMessageRepository.EXPECT().GetMessages(gomock.Any(), gomock.Any()).Return([]*model.Message{quoted}, false, nil).Times(1)

		result, _ := m.buildDetailedMessage(context.TODO(), mm, false, true, uuid.NewV3(uuid.Nil, "u1"))
		assert.Nil(t, result.Attachments)
		if assert.Len(t, result.Quotes, 1) {
			assert.Equal(t, quoteID, result.Quotes[0].ID)
			assert.Empty(t, result.Quotes[0].Attachments)
		}
	})

	t.Run("quotes with nested attachment", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mgr, _, repo, _ := setupM(ctrl)
		m := mgr.(*manager)

		quoteID := uuid.NewV3(uuid.Nil, "q1")
		nestedFileID := uuid.NewV3(uuid.Nil, "f1")
		userID := uuid.NewV3(uuid.Nil, "u1")
		mm := &model.Message{ID: uuid.NewV3(uuid.Nil, "m1"), Text: citationEmbed(quoteID)}
		quoted := &model.Message{ID: quoteID, Text: fileEmbed(nestedFileID)}

		repo.MockMessageRepository.EXPECT().GetMessages(gomock.Any(), gomock.Any()).Return([]*model.Message{quoted}, false, nil).Times(1)
		repo.MockFileRepository.EXPECT().IsFileAccessible(gomock.Any(), nestedFileID, userID).Return(true, nil).AnyTimes()
		repo.MockFileRepository.EXPECT().GetFileMeta(gomock.Any(), nestedFileID).Return(&model.FileMeta{ID: nestedFileID}, nil).Times(1)

		result, _ := m.buildDetailedMessage(context.TODO(), mm, false, true, uuid.NewV3(uuid.Nil, "u1"))
		if assert.Len(t, result.Quotes, 1) {
			if assert.Len(t, result.Quotes[0].Attachments, 1) {
				assert.Equal(t, nestedFileID, result.Quotes[0].Attachments[0].ID)
			}
		}
	})

	t.Run("GetMessages error for quotes results in empty quotes", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mgr, _, repo, _ := setupM(ctrl)
		m := mgr.(*manager)

		quoteID := uuid.NewV3(uuid.Nil, "q1")
		mm := &model.Message{ID: uuid.NewV3(uuid.Nil, "m1"), Text: citationEmbed(quoteID)}

		repo.MockMessageRepository.EXPECT().GetMessages(gomock.Any(), gomock.Any()).Return(nil, false, errors.New("db error")).Times(1)

		result, _ := m.buildDetailedMessage(context.TODO(), mm, false, true, uuid.NewV3(uuid.Nil, "u1"))
		assert.Empty(t, result.Quotes)
	})
}

func TestManager_GetTimeline_IncludeFlags(t *testing.T) {
	t.Parallel()

	t.Run("passes IncludeAttachments/IncludeQuotes through and resolves attachments", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		mgr, _, repo, _ := setupM(ctrl)

		cid := uuid.NewV3(uuid.Nil, "c1")
		fileID := uuid.NewV3(uuid.Nil, "f1")
		userID := uuid.NewV3(uuid.Nil, "u1")
		msg1 := &model.Message{
			ID:        uuid.NewV3(uuid.Nil, "m1"),
			ChannelID: cid,
			Text:      fileEmbed(fileID),
		}

		repo.MockMessageRepository.EXPECT().GetMessages(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, q repository.MessagesQuery) ([]*model.Message, bool, error) {
			assert.Equal(t, cid, q.Channel)
			assert.True(t, q.IncludeAttachments)
			assert.False(t, q.IncludeQuotes)
			return []*model.Message{msg1}, true, nil
		}).Times(1)

		repo.MockFileRepository.EXPECT().IsFileAccessible(gomock.Any(), fileID, userID).Return(true, nil).AnyTimes()
		repo.MockFileRepository.EXPECT().GetFileMeta(gomock.Any(), fileID).Return(&model.FileMeta{ID: fileID}, nil).Times(1)

		tl, err := mgr.GetTimeline(context.TODO(), TimelineQuery{
			Channel:            cid,
			IncludeAttachments: true,
			IncludeQuotes:      false,
		})
		if assert.NoError(t, err) {
			assert.True(t, tl.HasMore())
			records := tl.Records()
			if assert.Len(t, records, 1) {
				assert.Equal(t, msg1.ID, records[0].GetID())
			}
		}
	})
}
