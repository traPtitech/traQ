package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"

	"github.com/traPtitech/traQ/model"
)

func TestS3FileStorage(t *testing.T) {
	fs, err := NewS3FileStorage(bucketName, "ap-northeast-1", s3Main.endpoint,
		s3AccessKey, s3SecretKey, true, t.TempDir())
	require.NoError(t, err)

	for _, tc := range []struct {
		name string
		size int
	}{
		{name: "empty", size: 0},
		{name: "small", size: 1024},
		// Exceeds the transfer manager's 16 MiB multipart threshold.
		{name: "multipart", size: 17 * 1024 * 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key := "storage/" + tc.name
			data := bytes.Repeat([]byte("x"), tc.size)
			// Exercise the streaming reader used by file uploads.
			err := fs.SaveByKey(bytes.NewBuffer(data), key, "添付.txt", "text/plain", model.FileTypeUserFile)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, fs.DeleteByKey(key, model.FileTypeUserFile)) })

			metadata, err := s3Main.client.HeadObject(context.Background(), &s3.HeadObjectInput{
				Bucket: aws.String(bucketName), Key: aws.String(key),
			})
			require.NoError(t, err)
			require.Equal(t, "text/plain", aws.ToString(metadata.ContentType))
			require.Equal(t, "attachment; filename*=UTF-8''%E6%B7%BB%E4%BB%98.txt", aws.ToString(metadata.ContentDisposition))

			file, err := fs.OpenFileByKey(key, model.FileTypeUserFile)
			require.NoError(t, err)
			got, err := io.ReadAll(file)
			require.NoError(t, err)
			require.NoError(t, file.Close())
			require.Equal(t, data, got)

			url, err := fs.GenerateAccessURL(key, model.FileTypeUserFile)
			require.NoError(t, err)
			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Get(url)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode)
			got, err = io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, data, got)

			require.NoError(t, fs.DeleteByKey(key, model.FileTypeUserFile))
			_, err = fs.OpenFileByKey(key, model.FileTypeUserFile)
			require.ErrorIs(t, err, ErrFileNotFound)
		})
	}
}
