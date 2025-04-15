package keeper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Value copy from gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/transport
type Value struct {
	// Value хранит исходное значение параметра в сервисе keeper.
	Value string `json:"value"`
	// Version хранит текущую версию ключа. Удаление приводит к сбросу версии в 0.
	// Модификация значения приводит к увеличению данного значения.
	Version int64 `json:"version,omitempty"`
	// Время обновления записи
	ModRevision int64 `json:"mod_revision,omitempty"`
}

func (v *Value) Equal(v2 *Value) bool {
	if v == nil || v2 == nil {
		return false
	}

	return v.Version == v2.Version && v.ModRevision == v2.ModRevision
}

var badStatusErr = errors.New("bad response status from keeper")

// KeeperClient реализован свой кипер-клиент, по причине того что нельзя поднимать версию Go у пакета из за версии Go у traefik
type KeeperClient struct {
	client           *http.Client
	url              string
	settingsEndpoint string
	key              string
}

func NewKeeperClient(cl *http.Client, url, settingsEndpoint, key string) *KeeperClient {
	if settingsEndpoint == "" {
		settingsEndpoint = "admin/get"
	}

	return &KeeperClient{
		client:           cl,
		url:              url,
		settingsEndpoint: settingsEndpoint,
		key:              key,
	}
}

func (c *KeeperClient) GetRateLimits(ctx context.Context) (*Value, error) {
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
