package v3

import (
	"bytes"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
	"github.com/orcaman/writerseeker"
	"github.com/stretchr/testify/require"
	"github.com/traPtitech/traQ/model"
	"github.com/traPtitech/traQ/router/session"
	file2 "github.com/traPtitech/traQ/service/file"
	"math"
	"net/http"
	"testing"
)

func TestHandlers_GetFileMeta(t *testing.T) {
	t.Parallel()
	path := "/api/v3/files/{fileID}/meta"
	env := Setup(t, common1)
	s := env.S(t, env.CreateUser(t, rand).GetID())
	file := env.MakeFile(t)

	t.Run("not logged in", func(t *testing.T) {
		t.Parallel()

		e := env.R(t)
		e.GET(path, file.GetID()).
			Expect().
			Status(http.StatusUnauthorized)
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

		obj.Value("id").String().Equal(file.GetID().String())
		obj.Value("name").String().Equal(file.GetFileName())
		obj.Value("mime").String().Equal(file.GetMIMEType())
		obj.Value("size").Number().Equal(file.GetFileSize())
		obj.Value("md5").String().Equal(file.GetMD5Hash())
		obj.Value("thumbnails").Array().Length().Equal(0)
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

		obj.Value("id").String().Equal(iconFileID.String())
		thumbnails := obj.Value("thumbnails").Array()
		thumbnails.Length().Equal(1)
		thumbnail := thumbnails.First().Object()
		thumbnail.Value("type").Equal("image")
		thumbnail.Value("mime").Equal("image/png")
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

		obj.Value("id").String().Equal(file.GetID().String())
		thumbnails := obj.Value("thumbnails").Array()
		thumbnails.Length().Equal(1)
		thumbnail := thumbnails.First().Object()
		thumbnail.Value("type").Equal("waveform")
		thumbnail.Value("mime").Equal("image/svg+xml")
		thumbnail.Value("width").NotNull().NotEqual(0)
		thumbnail.Value("height").NotNull().NotEqual(0)
	})
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
