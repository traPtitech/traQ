package message

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/service/channel/mock_channel"
)

func setupM(ctrl *gomock.Controller) (Manager, *mock_channel.MockManager, *Repo, *mock_channel.MockTree) {
	cm := mock_channel.NewMockManager(ctrl)
	tree := mock_channel.NewMockTree(ctrl)
	cm.EXPECT().PublicChannelTree().Return(tree).AnyTimes()
	repo := NewMockRepo(ctrl)
	m, _ := NewMessageManager(repo, cm, zap.NewNop())
	return m, cm, repo, tree
}

func TestManager_Get(t *testing.T) {
	t.Parallel()

	t.Run("nil id", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, _, _, _ := setupM(ctrl)

		_, err := m.Get(uuid.Nil)
		assert.EqualError(t, err, ErrNotFound.Error())
	})

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, _, repo, _ := setupM(ctrl)

		msg := &model.Message{
			ID:        uuid.NewV3(uuid.Nil, "m1"),
			UserID:    uuid.NewV3(uuid.Nil, "u1"),
			ChannelID: uuid.NewV3(uuid.Nil, "c1"),
			Text:      "test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Stamps:    []model.MessageStamp{},
			Pin:       nil,
		}
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(msg.ID).
			Return(msg, nil).
			Times(1)

		result, err := m.Get(msg.ID)
		if assert.NoError(t, err) {
			assert.EqualValues(t, msg.ID, result.GetID())
			assert.EqualValues(t, msg.ChannelID, result.GetChannelID())
			assert.EqualValues(t, msg.UserID, result.GetUserID())
			assert.EqualValues(t, msg.Text, result.GetText())
			assert.EqualValues(t, msg.CreatedAt, result.GetCreatedAt())
			assert.EqualValues(t, msg.UpdatedAt, result.GetUpdatedAt())
			assert.EqualValues(t, msg.Pin, result.GetPin())
			assert.EqualValues(t, msg.Stamps, result.GetStamps())
		}

		// キャッシュからもう一度
		result, err = m.Get(msg.ID)
		if assert.NoError(t, err) {
			assert.EqualValues(t, msg.ID, result.GetID())
			assert.EqualValues(t, msg.ChannelID, result.GetChannelID())
			assert.EqualValues(t, msg.UserID, result.GetUserID())
			assert.EqualValues(t, msg.Text, result.GetText())
			assert.EqualValues(t, msg.CreatedAt, result.GetCreatedAt())
			assert.EqualValues(t, msg.UpdatedAt, result.GetUpdatedAt())
			assert.EqualValues(t, msg.Pin, result.GetPin())
			assert.EqualValues(t, msg.Stamps, result.GetStamps())
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, _, repo, _ := setupM(ctrl)

		id := uuid.NewV3(uuid.Nil, "m1")
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(id).
			Return(nil, repository.ErrNotFound).
			Times(1)

		_, err := m.Get(id)
		assert.EqualError(t, err, ErrNotFound.Error())
	})
}

func TestManager_Create(t *testing.T) {
	t.Parallel()
	const content = "content"

	t.Run("channel archived", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, cm, _, tree := setupM(ctrl)

		cid := uuid.NewV3(uuid.Nil, "c1")
		cm.EXPECT().IsPublicChannel(cid).Return(true).Times(1)
		tree.EXPECT().IsArchivedChannel(cid).Return(true).Times(1)

		_, err := m.Create(cid, uuid.NewV3(uuid.Nil, "u1"), content)
		assert.EqualError(t, err, ErrChannelArchived.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, cm, repo, tree := setupM(ctrl)

		cid := uuid.NewV3(uuid.Nil, "c1")
		uid := uuid.NewV3(uuid.Nil, "u1")
		cm.EXPECT().IsPublicChannel(cid).Return(true).Times(1)
		tree.EXPECT().IsArchivedChannel(cid).Return(false).Times(1)
		repo.MockMessageRepository.
			EXPECT().
			CreateMessage(uid, cid, content).
			Return(&model.Message{ID: uuid.NewV3(uuid.Nil, "m1"), UserID: uid, ChannelID: cid, Text: content}, nil).
			Times(1)

		msg, err := m.Create(cid, uid, content)
		if assert.NoError(t, err) {
			assert.EqualValues(t, cid, msg.GetChannelID())
			assert.EqualValues(t, uid, msg.GetUserID())
			assert.EqualValues(t, content, msg.GetText())
		}

		// キャッシュからもう一度
		result, err := m.Get(msg.GetID())
		if assert.NoError(t, err) {
			assert.EqualValues(t, msg.GetID(), result.GetID())
			assert.EqualValues(t, cid, result.GetChannelID())
			assert.EqualValues(t, uid, result.GetUserID())
			assert.EqualValues(t, content, result.GetText())
		}
	})
}

func TestManager_CreateDM(t *testing.T) {
	t.Parallel()
	const content = "content"

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, cm, repo, _ := setupM(ctrl)

		cid := uuid.NewV3(uuid.Nil, "c1")
		from := uuid.NewV3(uuid.Nil, "u1")
		to := uuid.NewV3(uuid.Nil, "u2")
		cm.EXPECT().GetDMChannel(from, to).Return(&model.Channel{ID: cid}, nil).Times(1)
		repo.MockMessageRepository.
			EXPECT().
			CreateMessage(from, cid, content).
			Return(&model.Message{ID: uuid.NewV3(uuid.Nil, "m1"), UserID: from, ChannelID: cid, Text: content}, nil).
			Times(1)

		msg, err := m.CreateDM(from, to, content)
		if assert.NoError(t, err) {
			assert.EqualValues(t, cid, msg.GetChannelID())
			assert.EqualValues(t, from, msg.GetUserID())
			assert.EqualValues(t, content, msg.GetText())
		}

		// キャッシュからもう一度
		result, err := m.Get(msg.GetID())
		if assert.NoError(t, err) {
			assert.EqualValues(t, msg.GetID(), result.GetID())
			assert.EqualValues(t, cid, result.GetChannelID())
			assert.EqualValues(t, from, result.GetUserID())
			assert.EqualValues(t, content, result.GetText())
		}
	})
}

func TestManager_Edit(t *testing.T) {
	t.Parallel()
	const newContent = "new message"

	t.Run("message not found", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, _, repo, _ := setupM(ctrl)

		id := uuid.NewV3(uuid.Nil, "m1")
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(id).
			Return(nil, repository.ErrNotFound).
			Times(1)

		err := m.Edit(id, newContent)
		assert.EqualError(t, err, ErrNotFound.Error())
	})

	t.Run("channel archived", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, cm, repo, tree := setupM(ctrl)

		id := uuid.NewV3(uuid.Nil, "m1")
		cid := uuid.NewV3(uuid.Nil, "c1")
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(id).
			Return(&model.Message{ID: id, ChannelID: cid}, nil).
			Times(1)
		cm.EXPECT().IsPublicChannel(cid).Return(true).Times(1)
		tree.EXPECT().IsArchivedChannel(cid).Return(true).Times(1)

		err := m.Edit(id, newContent)
		assert.EqualError(t, err, ErrChannelArchived.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, cm, repo, tree := setupM(ctrl)

		id := uuid.NewV3(uuid.Nil, "m1")
		cid := uuid.NewV3(uuid.Nil, "c1")
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(id).
			Return(&model.Message{ID: id, ChannelID: cid}, nil).
			Times(1)
		cm.EXPECT().IsPublicChannel(cid).Return(true).Times(1)
		tree.EXPECT().IsArchivedChannel(cid).Return(false).Times(1)
		repo.MockMessageRepository.
			EXPECT().
			UpdateMessage(id, newContent).
			Return(nil).
			Times(1)

		err := m.Edit(id, newContent)
		assert.NoError(t, err)
	})
}

func TestManager_Delete(t *testing.T) {
	t.Parallel()

	t.Run("message not found", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, _, repo, _ := setupM(ctrl)

		id := uuid.NewV3(uuid.Nil, "m1")
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(id).
			Return(nil, repository.ErrNotFound).
			Times(1)

		err := m.Delete(id)
		assert.EqualError(t, err, ErrNotFound.Error())
	})

	t.Run("channel archived", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, cm, repo, tree := setupM(ctrl)

		id := uuid.NewV3(uuid.Nil, "m1")
		cid := uuid.NewV3(uuid.Nil, "c1")
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(id).
			Return(&model.Message{ID: id, ChannelID: cid}, nil).
			Times(1)
		cm.EXPECT().IsPublicChannel(cid).Return(true).Times(1)
		tree.EXPECT().IsArchivedChannel(cid).Return(true).Times(1)

		err := m.Delete(id)
		assert.EqualError(t, err, ErrChannelArchived.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, cm, repo, tree := setupM(ctrl)

		id := uuid.NewV3(uuid.Nil, "m1")
		cid := uuid.NewV3(uuid.Nil, "c1")
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(id).
			Return(&model.Message{ID: id, ChannelID: cid}, nil).
			Times(1)
		cm.EXPECT().IsPublicChannel(cid).Return(true).Times(1)
		tree.EXPECT().IsArchivedChannel(cid).Return(false).Times(1)
		repo.MockMessageRepository.
			EXPECT().
			DeleteMessage(id).
			Return(nil).
			Times(1)

		err := m.Delete(id)
		assert.NoError(t, err)
	})
}

func TestManager_AddStamps(t *testing.T) {
	t.Parallel()

	t.Run("message not found", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, _, repo, _ := setupM(ctrl)

		id := uuid.NewV3(uuid.Nil, "m1")
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(id).
			Return(nil, repository.ErrNotFound).
			Times(1)

		_, err := m.AddStamps(id, uuid.NewV3(uuid.Nil, "s1"), uuid.NewV3(uuid.Nil, "u1"), 1)
		assert.EqualError(t, err, ErrNotFound.Error())
	})

	t.Run("channel archived", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, cm, repo, tree := setupM(ctrl)

		id := uuid.NewV3(uuid.Nil, "m1")
		cid := uuid.NewV3(uuid.Nil, "c1")
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(id).
			Return(&model.Message{ID: id, ChannelID: cid}, nil).
			Times(1)
		cm.EXPECT().IsPublicChannel(cid).Return(true).Times(1)
		tree.EXPECT().IsArchivedChannel(cid).Return(true).Times(1)

		_, err := m.AddStamps(id, uuid.NewV3(uuid.Nil, "s1"), uuid.NewV3(uuid.Nil, "u1"), 1)
		assert.EqualError(t, err, ErrChannelArchived.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, cm, repo, tree := setupM(ctrl)

		id := uuid.NewV3(uuid.Nil, "m1")
		cid := uuid.NewV3(uuid.Nil, "c1")
		sid := uuid.NewV3(uuid.Nil, "s1")
		sid2 := uuid.NewV3(uuid.Nil, "s2")
		uid := uuid.NewV3(uuid.Nil, "u1")
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(id).
			Return(&model.Message{ID: id, ChannelID: cid, Stamps: []model.MessageStamp{{
				MessageID: id,
				StampID:   sid2,
				UserID:    uid,
				Count:     10,
			}}}, nil).
			Times(1)
		cm.EXPECT().IsPublicChannel(cid).Return(true).Times(1)
		tree.EXPECT().IsArchivedChannel(cid).Return(false).Times(1)
		repo.MockMessageRepository.
			EXPECT().
			AddStampToMessage(id, sid, uid, 1).
			Return(&model.MessageStamp{
				MessageID: id,
				StampID:   sid,
				UserID:    uid,
				Count:     1,
			}, nil).
			Times(1)

		_, err := m.AddStamps(id, sid, uid, 1)
		if assert.NoError(t, err) {
			msg, err := m.Get(id)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, []model.MessageStamp{
					{
						MessageID: id,
						StampID:   sid,
						UserID:    uid,
						Count:     1,
					},
					{
						MessageID: id,
						StampID:   sid2,
						UserID:    uid,
						Count:     10,
					},
				}, msg.GetStamps())
			}
		}
	})
}

func TestManager_RemoveStamps(t *testing.T) {
	t.Parallel()

	t.Run("message not found", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, _, repo, _ := setupM(ctrl)

		id := uuid.NewV3(uuid.Nil, "m1")
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(id).
			Return(nil, repository.ErrNotFound).
			Times(1)

		err := m.RemoveStamps(id, uuid.NewV3(uuid.Nil, "s1"), uuid.NewV3(uuid.Nil, "u1"))
		assert.EqualError(t, err, ErrNotFound.Error())
	})

	t.Run("channel archived", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, cm, repo, tree := setupM(ctrl)

		id := uuid.NewV3(uuid.Nil, "m1")
		cid := uuid.NewV3(uuid.Nil, "c1")
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(id).
			Return(&model.Message{ID: id, ChannelID: cid}, nil).
			Times(1)
		cm.EXPECT().IsPublicChannel(cid).Return(true).Times(1)
		tree.EXPECT().IsArchivedChannel(cid).Return(true).Times(1)

		err := m.RemoveStamps(id, uuid.NewV3(uuid.Nil, "s1"), uuid.NewV3(uuid.Nil, "u1"))
		assert.EqualError(t, err, ErrChannelArchived.Error())
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		m, cm, repo, tree := setupM(ctrl)

		id := uuid.NewV3(uuid.Nil, "m1")
		cid := uuid.NewV3(uuid.Nil, "c1")
		sid := uuid.NewV3(uuid.Nil, "s1")
		sid2 := uuid.NewV3(uuid.Nil, "s2")
		uid := uuid.NewV3(uuid.Nil, "u1")
		repo.MockMessageRepository.
			EXPECT().
			GetMessageByID(id).
			Return(&model.Message{ID: id, ChannelID: cid, Stamps: []model.MessageStamp{
				{
					MessageID: id,
					StampID:   sid,
					UserID:    uid,
					Count:     1,
				},
				{
					MessageID: id,
					StampID:   sid2,
					UserID:    uid,
					Count:     10,
				},
			}}, nil).
			Times(1)
		cm.EXPECT().IsPublicChannel(cid).Return(true).Times(1)
		tree.EXPECT().IsArchivedChannel(cid).Return(false).Times(1)
		repo.MockMessageRepository.
			EXPECT().
			RemoveStampFromMessage(id, sid, uid).
			Return(nil).
			Times(1)

		err := m.RemoveStamps(id, sid, uid)
		if assert.NoError(t, err) {
			msg, err := m.Get(id)
			if assert.NoError(t, err) {
				assert.ElementsMatch(t, []model.MessageStamp{
					{
						MessageID: id,
						StampID:   sid2,
						UserID:    uid,
						Count:     10,
					},
				}, msg.GetStamps())
			}
		}
	})
}
