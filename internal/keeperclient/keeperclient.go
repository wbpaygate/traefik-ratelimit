package keeperclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var badStatusErr = errors.New("bad response status from keeper")

type Client struct {
	client           *http.Client
	url              string
	settingsEndpoint string
	key              string
}

func NewKeeperClient(url, settingsEndpoint, key string, timeout time.Duration) *Client {
	cl := &http.Client{
		Timeout: timeout,
	}

	if settingsEndpoint == "" {
		settingsEndpoint = "admin/get"
	}

	return &Client{
		client:           cl,
		url:              url,
		settingsEndpoint: settingsEndpoint,
		key:              key,
	}
}

func (c *Client) GetRateLimits(ctx context.Context) (*Value, error) {
	reqURL := fmt.Sprintf("%s/%s/%s", c.url, c.settingsEndpoint, c.key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http client do fail: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", badStatusErr, resp.StatusCode)
	}

	value := new(Value)

	err = json.NewDecoder(resp.Body).Decode(value)
	if err != nil {
		return nil, fmt.Errorf("cannot decode keeper response: %w", err)
	}

	return value, nil
}
