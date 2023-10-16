package rclonecli

import (
	"context"
	"net/http"
	"testing"

	"github.com/steinfletcher/apitest"
	"github.com/stretchr/testify/assert"
)

func TestRefreshCache(t *testing.T) {
	host := "http://localhost:5572"
	hc := &http.Client{}

	defer apitest.NewMock().
		HttpClient(hc).
		Postf("%s/vfs/refresh", host).
		Header("Content-Type", "application/json").
		Body(`{"_async":"true","recursive":"false"}`).
		RespondWith().
		Status(http.StatusOK).
		Body(`{"status":"ok"}`).
		EndStandalone()()

	client := NewRcloneRcClient(host, hc)

	err := client.RefreshCache(context.Background(), "", true, false)
	assert.NoError(t, err)
}

func TestRefreshCacheWithDir(t *testing.T) {
	host := "http://localhost:5572"
	hc := &http.Client{}

	defer apitest.NewMock().
		HttpClient(hc).
		Postf("%s/vfs/refresh", host).
		Header("Content-Type", "application/json").
		Body(`{"_async":"true","dir":"/foo","recursive":"false"}`).
		RespondWith().
		Status(http.StatusOK).
		Body(`{"status":"ok"}`).
		EndStandalone()()

	client := NewRcloneRcClient(host, hc)

	err := client.RefreshCache(context.Background(), "/foo", true, false)
	assert.NoError(t, err)
}

func TestRefreshCacheWithDirAndRecursive(t *testing.T) {
	host := "http://localhost:5572"
	hc := &http.Client{}

	defer apitest.NewMock().
		HttpClient(hc).
		Postf("%s/vfs/refresh", host).
		Header("Content-Type", "application/json").
		Body(`{"_async":"true","dir":"/foo","recursive":"true"}`).
		RespondWith().
		Status(http.StatusOK).
		Body(`{"status":"ok"}`).
		EndStandalone()()

	client := NewRcloneRcClient(host, hc)

	err := client.RefreshCache(context.Background(), "/foo", true, true)
	assert.NoError(t, err)
}

func TestRefreshCacheWithDirAndRecursiveAndNotAsync(t *testing.T) {
	host := "http://localhost:5572"
	hc := &http.Client{}

	defer apitest.NewMock().
		HttpClient(hc).
		Postf("%s/vfs/refresh", host).
		Header("Content-Type", "application/json").
		Body(`{"_async":"false","dir":"/foo","recursive":"true"}`).
		RespondWith().
		Status(http.StatusOK).
		Body(`{"status":"ok"}`).
		EndStandalone()()

	client := NewRcloneRcClient(host, hc)

	err := client.RefreshCache(context.Background(), "/foo", false, true)
	assert.NoError(t, err)
}

func TestRefreshCacheWithError(t *testing.T) {
	host := "http://localhost:5572"
	hc := &http.Client{}

	defer apitest.NewMock().
		HttpClient(hc).
		Postf("%s/vfs/refresh", host).
		Header("Content-Type", "application/json").
		Body(`{"_async":"false","dir":"/foo","recursive":"true"}`).
		RespondWith().
		Status(http.StatusInternalServerError).
		Body(`{"error":"error"}`).
		EndStandalone()()

	client := NewRcloneRcClient(host, hc)

	err := client.RefreshCache(context.Background(), "/foo", false, true)
	assert.Error(t, err)
}
