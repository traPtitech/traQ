package v3

import (
	"bytes"
	"context"
	"encoding/binary"
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/gofrs/uuid"
	"github.com/livekit/protocol/livekit"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/repository"
	"github.com/traPtitech/traQ/router/session"
)

func TestHandlers_GetQallEndpoints(t *testing.T) {
	t.Parallel()

	path := "/api/v3/qall/endpoints"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	client := env.CreateOAuth2Client(t, rand, user.GetID())
	commonSession := env.S(t, user.GetID())

	issueToken := func(t *testing.T, scopes ...model.AccessScope) string {
		t.Helper()
		s := model.AccessScopes{}
		s.Add(scopes...)
		tok, err := env.Repository.IssueToken(context.TODO(), client, user.GetID(), client.RedirectURI, s, 86400, false)
		require.NoError(t, err)
		return tok.AccessToken
	}

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success (session)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithCookie(session.CookieName, commonSession).
			Expect().
			Status(http.StatusOK)
	})

	t.Run("forbidden (oauth2 token without qall scope)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithHeader("Authorization", "Bearer "+issueToken(t, "read", "write")).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("success (oauth2 token with qall scope)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithHeader("Authorization", "Bearer "+issueToken(t, "read", "qall")).
			Expect().
			Status(http.StatusOK)
	})
}

// silentWav sampleRate Hz・モノラル・16bit の無音の WAV を返す
func silentWav(t *testing.T, seconds int) []byte {
	t.Helper()
	const sampleRate = 8000
	dataSize := sampleRate * 2 * seconds
	buf := &bytes.Buffer{}
	write := func(v any) {
		require.NoError(t, binary.Write(buf, binary.LittleEndian, v))
	}
	buf.WriteString("RIFF")
	write(uint32(36 + dataSize))
	buf.WriteString("WAVEfmt ")
	write(uint32(16))
	write(uint16(1)) // PCM
	write(uint16(1)) // mono
	write(uint32(sampleRate))
	write(uint32(sampleRate * 2))
	write(uint16(2))
	write(uint16(16))
	buf.WriteString("data")
	write(uint32(dataSize))
	buf.Write(make([]byte, dataSize))
	return buf.Bytes()
}

func TestHandlers_CreateSoundboardItem(t *testing.T) {
	t.Parallel()

	path := "/api/v3/qall/soundboard"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	other := env.CreateUser(t, rand)
	s := env.S(t, user.GetID())
	wavHeader := func(e *httpexpect.Request, field string, data []byte) *httpexpect.Request {
		return e.WithMultipart().WithFile(field, "sound.wav", bytes.NewReader(data))
	}

	findItem := func(t *testing.T, soundID string) *httpexpect.Object {
		t.Helper()
		arr := env.R(t).GET(path).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()
		for _, v := range arr.Iter() {
			obj := v.Object()
			if obj.Value("soundId").String().Raw() == soundID {
				return obj
			}
		}
		t.Fatalf("sound %s was not found in the list", soundID)
		return nil
	}

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		env.R(t).POST(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("success with fields in the API definition", func(t *testing.T) {
		t.Parallel()
		req := env.R(t).POST(path).WithCookie(session.CookieName, s)
		soundID := wavHeader(req, "audio", silentWav(t, 1)).
			WithFormField("soundName", "spec").
			Expect().
			Status(http.StatusOK).
			JSON().
			Object().
			Value("soundId").
			String().
			Raw()

		obj := findItem(t, soundID)
		obj.Value("soundName").IsEqual("spec")
		obj.Value("creatorId").IsEqual(user.GetID().String())
		obj.Value("stampId").IsNull()
		// 後方互換のフィールド
		obj.Value("id").IsEqual(soundID)
		obj.Value("name").IsEqual("spec")
	})

	t.Run("success with legacy fields", func(t *testing.T) {
		t.Parallel()
		stamp := env.CreateStamp(t, user.GetID(), rand)
		req := env.R(t).POST(path).WithCookie(session.CookieName, s)
		soundID := wavHeader(req, "file", silentWav(t, 1)).
			WithFormField("name", "legacy").
			WithFormField("stampId", stamp.ID.String()).
			// 作成者は指定しても無視され、リクエストしたユーザーになる
			WithFormField("creatorId", other.GetID().String()).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object().
			Value("soundId").
			String().
			Raw()

		obj := findItem(t, soundID)
		obj.Value("soundName").IsEqual("legacy")
		obj.Value("stampId").IsEqual(stamp.ID.String())
		obj.Value("creatorId").IsEqual(user.GetID().String())
	})

	t.Run("bad request (no file)", func(t *testing.T) {
		t.Parallel()
		env.R(t).POST(path).
			WithCookie(session.CookieName, s).
			WithMultipart().
			WithFormField("soundName", "nofile").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (no name)", func(t *testing.T) {
		t.Parallel()
		req := env.R(t).POST(path).WithCookie(session.CookieName, s)
		wavHeader(req, "audio", silentWav(t, 1)).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (invalid stamp id)", func(t *testing.T) {
		t.Parallel()
		req := env.R(t).POST(path).WithCookie(session.CookieName, s)
		wavHeader(req, "audio", silentWav(t, 1)).
			WithFormField("soundName", "badstamp").
			WithFormField("stampId", "not-a-uuid").
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (too long)", func(t *testing.T) {
		t.Parallel()
		req := env.R(t).POST(path).WithCookie(session.CookieName, s)
		wavHeader(req, "audio", silentWav(t, 31)).
			WithFormField("soundName", "long").
			Expect().
			Status(http.StatusBadRequest)
	})
}

func TestHandlers_PlaySoundboardItem(t *testing.T) {
	t.Parallel()

	path := "/api/v3/qall/soundboard/play"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	outsider := env.CreateUser(t, rand)
	s := env.S(t, user.GetID())
	outsiderSession := env.S(t, outsider.GetID())

	soundID := uuid.Must(uuid.NewV7())
	require.NoError(t, env.Repository.CreateSoundboardItem(context.TODO(), repository.CreateSoundboardItemArgs{
		SoundID:   soundID,
		SoundName: "play",
		CreatorID: user.GetID(),
	}))

	roomID := uuid.Must(uuid.NewV4())
	env.QallRepo.AddParticipantToRoomState(
		&livekit.Room{Name: roomID.String()},
		&livekit.ParticipantInfo{Identity: user.GetID().String() + "_" + uuid.Must(uuid.NewV7()).String(), Name: user.GetID().String()},
	)

	t.Run("bad request (invalid room id)", func(t *testing.T) {
		t.Parallel()
		env.R(t).POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]string{"soundId": soundID.String()}).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("not found (unknown sound)", func(t *testing.T) {
		t.Parallel()
		env.R(t).POST(path).
			WithCookie(session.CookieName, s).
			WithJSON(map[string]string{"soundId": uuid.Must(uuid.NewV7()).String(), "roomName": roomID.String()}).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("forbidden (not a participant)", func(t *testing.T) {
		t.Parallel()
		env.R(t).POST(path).
			WithCookie(session.CookieName, outsiderSession).
			WithJSON(map[string]string{"soundId": soundID.String(), "roomName": roomID.String()}).
			Expect().
			Status(http.StatusForbidden)
	})

	// テスト環境のストレージは署名付き URL を発行できないので、参加者の確認を通過したところで 500 になる
	for _, field := range []string{"roomName", "roomId"} {
		t.Run("accepts "+field, func(t *testing.T) {
			t.Parallel()
			env.R(t).POST(path).
				WithCookie(session.CookieName, s).
				WithJSON(map[string]string{"soundId": soundID.String(), field: roomID.String()}).
				Expect().
				Status(http.StatusInternalServerError)
		})
	}
}
