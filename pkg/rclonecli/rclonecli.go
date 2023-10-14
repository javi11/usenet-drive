package rclonecli

//go:generate mockgen -source=./rclonecli.go -destination=./rclonecli_mock.go -package=rclonecli RcloneRcClient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type RcloneRcClient interface {
	RefreshCache(ctx context.Context, dir string, async, recursive bool) error
}

type rcloneRcClient struct {
	url        string
	httpClient *http.Client
}

func NewRcloneRcClient(url string, httpClient *http.Client) RcloneRcClient {
	return &rcloneRcClient{url: url, httpClient: httpClient}
}

func (c *rcloneRcClient) RefreshCache(ctx context.Context, dir string, async, recursive bool) error {
	data := map[string]string{
		"_async":    fmt.Sprintf("%t", async),
		"recursive": fmt.Sprintf("%t", recursive),
	}

	if dir != "" {
		data["dir"] = dir
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.url+"/vfs/refresh", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("unexpected status code: %d, error: %s", resp.StatusCode, string(body))
	}

	return nil
}
