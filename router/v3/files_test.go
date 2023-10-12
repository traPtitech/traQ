package v3

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/gavv/httpexpect/v2"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/gofrs/uuid"
	"github.com/labstack/echo/v4"
	"github.com/orcaman/writerseeker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/session"
	file2 "github.com/traPtitech/traQ/service/file"
	"github.com/traPtitech/traQ/utils/optional"
)

func fileEquals(t *testing.T, expect model.File, actual *httpexpect.Object) {
	t.Helper()
	actual.Value("id").String().IsEqual(expect.GetID().String())
	actual.Value("name").String().IsEqual(expect.GetFileName())
	actual.Value("mime").String().IsEqual(expect.GetMIMEType())
	actual.Value("size").Number().IsEqual(expect.GetFileSize())
	actual.Value("md5").String().IsEqual(expect.GetMD5Hash())
	actual.Value("isAnimatedImage").Boolean().IsEqual(expect.IsAnimatedImage())
	actual.Value("thumbnails").Array().Length().IsEqual(len(expect.GetThumbnails()))
	actual.Value("channelId").IsEqual(expect.GetUploadChannelID())
	actual.Value("uploaderId").IsEqual(expect.GetCreatorID())
}

func genSampleAudio(t *testing.T) *bytes.Reader {
	t.Helper()

	const (
		sampleRate = 48000
		bitDepth   = 16
		length     = 0.1
	)

	sinWave := func(amplitude, frequency, t float64) float64 {
		return amplitude * math.Sin(frequency*2*math.Pi*t)
	}

	buf := &writerseeker.WriterSeeker{}
	e := wav.NewEncoder(buf, sampleRate, bitDepth, 1, 1)

	data := make([]int, sampleRate*length)
	for i := 0; i < sampleRate*length; i++ {
		data[i] = int(sinWave(math.Pow(2, 15)-1, 440, float64(i)/sampleRate))
	}

	require.NoError(t, e.Write(&audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: 1,
			SampleRate:  sampleRate,
		},
		Data:           data,
		SourceBitDepth: bitDepth,
	}))
	require.NoError(t, e.Close())

	return buf.BytesReader()
}

func TestGetFilesRequest_Validate(t *testing.T) {
	type fields struct {
		Limit     int
		Offset    int
		Since     optional.Of[time.Time]
		Until     optional.Of[time.Time]
		Inclusive bool
		Order     string
		ChannelID uuid.UUID
		Mine      bool
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"zero limit",
			fields{Limit: 0, Mine: true},
			false,
		},
		{
			"too large limit",
			fields{Limit: 500, Mine: true},
			true,
		},
		{
			"neither mine or in specific channel",
			fields{},
			true,
		},
		{
			"in channel",
			fields{ChannelID: uuid.Must(uuid.NewV4())},
			false,
		},
		{
			"mine",
			fields{Mine: true},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &GetFilesRequest{
				Limit:     tt.fields.Limit,
				Offset:    tt.fields.Offset,
				Since:     tt.fields.Since,
				Until:     tt.fields.Until,
				Inclusive: tt.fields.Inclusive,
				Order:     tt.fields.Order,
				ChannelID: tt.fields.ChannelID,
				Mine:      tt.fields.Mine,
			}
			if err := q.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandlers_GetFiles(t *testing.T) {
	t.Parallel()

	path := "/api/v3/files"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	dm := env.CreateDMChannel(t, user2.GetID(), user3.GetID())
	f1 := env.CreateFile(t, user.GetID(), uuid.Nil)
	f2 := env.CreateFile(t, uuid.Nil, ch.ID)
	env.CreateFile(t, uuid.Nil, dm.ID)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path).
			WithCookie(session.CookieName, s).
			WithQuery("channelId", dm.ID.String()).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success (mine)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, s).
			WithQuery("mine", true).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)
		fileEquals(t, f1, obj.Value(0).Object())
	})

	t.Run("success (channel)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path).
			WithCookie(session.CookieName, s).
			WithQuery("channelId", ch.ID.String()).
			Expect().
			Status(http.StatusOK).
			JSON().
			Array()

		obj.Length().IsEqual(1)
		fileEquals(t, f2, obj.Value(0).Object())
	})
}

func TestHandlers_PostFile(t *testing.T) {
	t.Parallel()

	path := "/api/v3/files"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	ch := env.CreateChannel(t, rand)
	dm1 := env.CreateDMChannel(t, user.GetID(), user2.GetID())
	dm2 := env.CreateDMChannel(t, user2.GetID(), user3.GetID())
	archived := env.CreateChannel(t, rand)
	require.NoError(t, env.CM.ArchiveChannel(archived.ID, user.GetID()))
	s := env.S(t, user.GetID())

	buf := []byte("test file")
	sum := md5.Sum(buf)
	hexSum := hex.EncodeToString(sum[:])

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithMultipart().
			WithFileBytes("file", "file.txt", buf).
			WithFormField("channelId", ch.ID.String()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("bad request (no file)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithMultipart().
			WithFormField("channelId", ch.ID.String()).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (no channel id)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithMultipart().
			WithFileBytes("file", "file.txt", buf).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithMultipart().
			WithFileBytes("file", "file.txt", buf).
			WithFormField("channelId", dm2.ID.String()).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("bad request (archived)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.POST(path).
			WithCookie(session.CookieName, s).
			WithMultipart().
			WithFileBytes("file", "file.txt", buf).
			WithFormField("channelId", archived.ID.String()).
			Expect().
			Status(http.StatusBadRequest)
	})

	t.Run("success (public)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path).
			WithCookie(session.CookieName, s).
			WithMultipart().
			WithFileBytes("file", "file.txt", buf).
			WithFormField("channelId", ch.ID.String()).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("name").String().IsEqual("file.txt")
		obj.Value("mime").String().NotEmpty()
		obj.Value("size").Number().IsEqual(9)
		obj.Value("md5").String().IsEqual(hexSum)
		obj.Value("isAnimatedImage").Boolean().IsFalse()
		obj.Value("thumbnails").Array().Length().IsEqual(0)
		obj.Value("channelId").String().IsEqual(ch.ID.String())
		obj.Value("uploaderId").String().IsEqual(user.GetID().String())
	})

	t.Run("success (dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.POST(path).
			WithCookie(session.CookieName, s).
			WithMultipart().
			WithFileBytes("file", "file.txt", buf).
			WithFormField("channelId", dm1.ID.String()).
			Expect().
			Status(http.StatusCreated).
			JSON().
			Object()

		obj.Value("id").String().NotEmpty()
		obj.Value("name").String().IsEqual("file.txt")
		obj.Value("mime").String().NotEmpty()
		obj.Value("size").Number().IsEqual(9)
		obj.Value("md5").String().IsEqual(hexSum)
		obj.Value("isAnimatedImage").Boolean().IsFalse()
		obj.Value("thumbnails").Array().Length().IsEqual(0)
		obj.Value("channelId").String().IsEqual(dm1.ID.String())
		obj.Value("uploaderId").String().IsEqual(user.GetID().String())
	})
}

func TestHandlers_GetFileMeta(t *testing.T) {
	t.Parallel()

	path := "/api/v3/files/{fileId}/meta"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	dm := env.CreateDMChannel(t, user2.GetID(), user3.GetID())
	s := env.S(t, user.GetID())
	file := env.CreateFile(t, user.GetID(), uuid.Nil)
	secretFile := env.CreateFile(t, user2.GetID(), dm.ID)

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, file.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, secretFile.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		obj := e.GET(path, file.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		fileEquals(t, file, obj)
	})

	t.Run("success with image thumbnail", func(t *testing.T) {
		t.Parallel()

		iconFileID, err := file2.GenerateIconFile(env.FM, "test")
		require.NoError(t, err)

		e := env.R(t)
		obj := e.GET(path, iconFileID).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("id").String().IsEqual(iconFileID.String())
		thumbnails := obj.Value("thumbnails").Array()
		thumbnails.Length().IsEqual(1)
		thumbnail := thumbnails.Value(0).Object()
		thumbnail.Value("type").IsEqual("image")
		thumbnail.Value("mime").IsEqual("image/png")
		thumbnail.Value("width").NotNull().NotEqual(0)
		thumbnail.Value("height").NotNull().NotEqual(0)
	})

	t.Run("success with waveform thumbnail", func(t *testing.T) {
		t.Parallel()

		sampleAudio := genSampleAudio(t)
		file, err := env.FM.Save(file2.SaveArgs{
			FileName: "sample.wav",
			FileSize: sampleAudio.Size(),
			MimeType: "audio/wav",
			FileType: model.FileTypeUserFile,
			Src:      sampleAudio,
		})
		require.NoError(t, err)

		e := env.R(t)
		obj := e.GET(path, file.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			JSON().
			Object()

		obj.Value("id").String().IsEqual(file.GetID().String())
		thumbnails := obj.Value("thumbnails").Array()
		thumbnails.Length().IsEqual(1)
		thumbnail := thumbnails.Value(0).Object()
		thumbnail.Value("type").IsEqual("waveform")
		thumbnail.Value("mime").IsEqual("image/svg+xml")
		thumbnail.Value("width").NotNull().NotEqual(0)
		thumbnail.Value("height").NotNull().NotEqual(0)
	})
}

func TestHandlers_GetThumbnailImage(t *testing.T) {
	t.Parallel()

	path := "/api/v3/files/{fileId}/thumbnail"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	f := env.CreateFile(t, user.GetID(), uuid.Nil)
	require.Len(t, f.GetThumbnails(), 0)

	dm := env.CreateDMChannel(t, user2.GetID(), user3.GetID())
	s := env.S(t, user.GetID())

	iconFile, err := file2.GenerateIconFile(env.FM, "test")
	require.NoError(t, err)

	sampleAudio := genSampleAudio(t)
	audioFile, err := env.FM.Save(file2.SaveArgs{
		FileName: "sample.wav",
		FileSize: sampleAudio.Size(),
		MimeType: "audio/wav",
		FileType: model.FileTypeUserFile,
		Src:      sampleAudio,
	})
	require.NoError(t, err)

	secretFile := env.CreateFile(t, user2.GetID(), dm.ID)

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, iconFile).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, secretFile.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("thumbnail not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, f.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success (type=image)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, iconFile).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			HasContentType("image/png")
	})

	t.Run("success (type=waveform)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, audioFile.GetID()).
			WithCookie(session.CookieName, s).
			WithQuery("type", "waveform").
			Expect().
			Status(http.StatusOK).
			HasContentType("image/svg+xml")
	})
}

func TestHandlers_GetFile(t *testing.T) {
	t.Parallel()

	path := "/api/v3/files/{fileId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	f := env.CreateFile(t, user.GetID(), uuid.Nil)
	f2 := env.CreateFileWithName(t, user.GetID(), uuid.Nil, "テス,ト")
	dm := env.CreateDMChannel(t, user2.GetID(), user3.GetID())
	secretFile := env.CreateFile(t, user2.GetID(), dm.ID)
	s := env.S(t, user.GetID())

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, f.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, secretFile.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, f.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			Body().
			IsEqual("test message")
	})

	t.Run("success Content-Dispotion escape", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.GET(path, f2.GetID()).
			WithQuery("dl", "1").
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusOK).
			Header(echo.HeaderContentDisposition).
			IsEqual("attachment; filename*=UTF-8''%E3%83%86%E3%82%B9%2C%E3%83%88")
	})
}

func TestHandlers_DeleteFile(t *testing.T) {
	t.Parallel()

	path := "/api/v3/files/{fileId}"
	env := Setup(t, common1)
	user := env.CreateUser(t, rand)
	user2 := env.CreateUser(t, rand)
	user3 := env.CreateUser(t, rand)
	f := env.CreateFile(t, user.GetID(), uuid.Nil)
	f2 := env.CreateFile(t, user2.GetID(), uuid.Nil)
	dm := env.CreateDMChannel(t, user2.GetID(), user3.GetID())
	secretFile := env.CreateFile(t, user2.GetID(), dm.ID)
	s := env.S(t, user.GetID())

	iconFile, err := file2.GenerateIconFile(env.FM, "test")
	require.NoError(t, err)

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, f.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
	})

	t.Run("forbidden (dm)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, secretFile.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("forbidden (different owner)", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, f2.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("cannot delete non user file", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, iconFile).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusForbidden)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, uuid.Must(uuid.NewV4())).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNotFound)
	})

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		e := env.R(t)
		e.DELETE(path, f.GetID()).
			WithCookie(session.CookieName, s).
			Expect().
			Status(http.StatusNoContent)

		_, err := env.FM.Get(f.GetID())
		assert.ErrorIs(t, err, file2.ErrNotFound)
	})
}
