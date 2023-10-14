package rclonecli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestRefreshCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTPClient := NewMockHTTPClient(ctrl)

	client := NewRcloneRcClient("http://localhost:5572", mockHTTPClient)

	ctx := context.Background()

	dir := "/path/to/dir"
	async := true
	recursive := false

	expectedData := map[string]string{
		"_async":    "true",
		"recursive": "false",
		"dir":       dir,
	}

	expectedPayload, err := json.Marshal(expectedData)
	assert.NoError(t, err)

	expectedResp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(bytes.NewBufferString("")),
	}

	mockHTTPClient.EXPECT().Do(gomock.Any()).DoAndReturn(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "http://localhost:5572/vfs/refresh", req.URL.String())

		body, err := ioutil.ReadAll(req.Body)
		assert.NoError(t, err)
		assert.Equal(t, expectedPayload, body)

		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

		return expectedResp, nil
	})

	err = client.RefreshCache(ctx, dir, async, recursive)
	assert.NoError(t, err)
}

func TestRefreshCache_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockHTTPClient := NewMockHTTPClient(ctrl)

	client := NewRcloneRcClient("http://localhost:5572", mockHTTPClient)

	ctx := context.Background()

	dir := "/path/to/dir"
	async := true
	recursive := false

	expectedData := map[string]string{
		"_async":    "true",
		"recursive": "false",
		"dir":       dir,
	}

	expectedPayload, err := json.Marshal(expectedData)
	assert.NoError(t, err)

	expectedResp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       ioutil.NopCloser(bytes.NewBufferString("error message")),
	}

	mockHTTPClient.EXPECT().Do(gomock.Any()).DoAndReturn(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "http://localhost:5572/vfs/refresh", req.URL.String())

		body, err := ioutil.ReadAll(req.Body)
		assert.NoError(t, err)
		assert.Equal(t, expectedPayload, body)

		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))

		return expectedResp, nil
	})

	err = client.RefreshCache(ctx, dir, async, recursive)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrUnexpectedStatusCode))
}
