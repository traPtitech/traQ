package router

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestPostFile(t *testing.T) {
	e, cookie, mw := beforeTest(t)
	assert := assert.New(t)

	body := createFormFile(t)

	req := httptest.NewRequest("POST", "http://test", body)
	req.Header.Set("Content-Type", "multipart/form-data")
	rec := request(e, t, mw(PostFile), cookie, req)

	assert.Equal(http.StatusCreated, rec.Code, rec.Body.String())
}

func TestGetFileByID(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	file := mustMakeFile(t)

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:fileID")
	c.SetParamNames("fileID")
	c.SetParamValues(file.ID)

	requestWithContext(t, mw(GetFileByID), c)
	if assert.EqualValues(t, http.StatusOK, rec.Code) {
		assert.Equal(t, "test message", rec.Body.String())
	}
}
func TestDeleteFileByID(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	file := mustMakeFile(t)

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:fileID")
	c.SetParamNames("fileID")
	c.SetParamValues(file.ID)

	requestWithContext(t, mw(DeleteFileByID), c)
	if assert.EqualValues(t, http.StatusNoContent, rec.Code, rec.Body.String()) {
		t.Log(rec.Body.String())
	}

}
func TestGetMetaDataByFileID(t *testing.T) {
	e, cookie, mw := beforeTest(t)

	file := mustMakeFile(t)

	c, rec := getContext(e, t, cookie, nil)
	c.SetPath("/:fileID")
	c.SetParamNames("fileID")
	c.SetParamValues(file.ID)

	requestWithContext(t, mw(GetMetaDataByFileID), c)
	if assert.EqualValues(t, http.StatusOK, rec.Code, rec.Body.String()) {
		t.Log(rec.Body.String())
	}
}

func createFormFile(t *testing.T) *bytes.Buffer {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("file", "test.txt")
	require.NoError(t, err)

	fh, err := os.Open("../LICENSE")
	require.NoError(t, err)
	defer fh.Close()

	_, err = io.Copy(fileWriter, fh)
	require.NoError(t, err)

	bodyWriter.Close()
	return bodyBuf
}
