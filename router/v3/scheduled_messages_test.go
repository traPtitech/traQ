package v3

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
	"github.com/traPtitech/traQ/service/rbac"
	"github.com/traPtitech/traQ/service/scheduled"
	"github.com/traPtitech/traQ/utils/optional"
)

func TestScheduledMessageRequestValidation(t *testing.T) {
	valid := PostScheduledMessageRequest{ChannelID: uuid.Must(uuid.NewV7()), Content: "notice", ScheduledAt: time.Now().Add(time.Hour)}
	require.NoError(t, valid.Validate())
	for _, name := range []string{"missing channel", "past", "empty", "too long", "duplicate file", "nil file"} {
		t.Run(name, func(t *testing.T) {
			r := valid
			switch name {
			case "missing channel":
				r.ChannelID = uuid.Nil
			case "past":
				r.ScheduledAt = time.Now().Add(-time.Second)
			case "empty":
				r.Content = ""
			case "too long":
				r.Content = strings.Repeat("あ", 10001)
			case "duplicate file":
				id := uuid.Must(uuid.NewV7())
				r.FileIDs = []uuid.UUID{id, id}
			case "nil file":
				r.FileIDs = []uuid.UUID{uuid.Nil}
			}
			require.Error(t, r.Validate())
		})
	}
}

func TestScheduledMessageAPIPrivateAttachmentsAndDMDelivery(t *testing.T) {
	ctx := context.Background()
	env := Setup(t, common1)
	user, other, outsider := env.CreateUser(t, rand), env.CreateUser(t, rand), env.CreateUser(t, rand)
	dm, err := env.CM.GetDMChannel(ctx, user.GetID(), other.GetID())
	require.NoError(t, err)
	s := env.S(t, user.GetID())
	otherSession := env.S(t, other.GetID())
	path := "/api/v3/users/me/scheduled-messages"
	e := env.R(t)
	e.GET(path).Expect().Status(http.StatusUnauthorized)
	fidString := e.POST(path+"/files").WithCookie(session.CookieName, s).
		WithMultipart().WithFormField("channelId", dm.ID.String()).
		WithFile("file", "notice.txt", bytes.NewBufferString("private announcement")).
		Expect().Status(http.StatusCreated).JSON().Object().Value("id").String().Raw()
	fid := uuid.FromStringOrNil(fidString)
	obj := e.POST(path).WithCookie(session.CookieName, s).WithJSON(PostScheduledMessageRequest{
		ChannelID: dm.ID, Content: "notice /files/" + fidString, DraftContent: "notice",
		ScheduledAt: time.Now().Add(time.Hour), FileIDs: []uuid.UUID{fid},
	}).Expect().Status(http.StatusCreated).JSON().Object()
	idString := obj.Value("id").String().Raw()
	id := uuid.FromStringOrNil(idString)
	obj.Value("status").String().IsEqual("pending")
	obj.Value("draftContent").String().IsEqual("notice")
	e.GET(path).WithCookie(session.CookieName, otherSession).Expect().Status(http.StatusOK).JSON().Array().IsEmpty()
	e.DELETE(path+"/"+idString).WithCookie(session.CookieName, otherSession).Expect().Status(http.StatusNotFound)
	visible, err := env.Repository.IsFileAccessible(ctx, fid, other.GetID())
	require.NoError(t, err)
	require.False(t, visible)
	files, _, err := env.Repository.GetFileMetas(ctx, repository.FilesQuery{ChannelID: optional.From(dm.ID), Type: model.FileTypeUserFile})
	require.NoError(t, err)
	require.Empty(t, files)
	// Simulate recovery after the scheduled time without waiting for a clock tick.
	require.NoError(t, env.DB.Model(&model.ScheduledMessage{}).Where("id = ?", id).Update("scheduled_at", time.Now().Add(-time.Hour)).Error)
	access, err := rbac.New(env.Repository)
	require.NoError(t, err)
	w := scheduled.Worker{Repo: env.Repository, Channels: env.CM, RBAC: access, Logger: zap.NewNop()}
	require.NoError(t, w.DeliverDue(ctx))
	visible, err = env.Repository.IsFileAccessible(ctx, fid, other.GetID())
	require.NoError(t, err)
	require.True(t, visible)
	visible, err = env.Repository.IsFileAccessible(ctx, fid, outsider.GetID())
	require.NoError(t, err)
	require.False(t, visible)
	e.DELETE(path+"/"+idString).WithCookie(session.CookieName, s).Expect().Status(http.StatusConflict)
	e.GET(path).WithCookie(session.CookieName, s).Expect().Status(http.StatusOK).JSON().Array().IsEmpty()
}
