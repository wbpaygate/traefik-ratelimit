package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

var _ Transport = (*HTTPTransport)(nil)

type HTTPTransport struct {
	keeperURL              string
	getSettingsEndpoint    string
	getAllSettingsEndpoint string
	httpClient             *http.Client
}

func (t *HTTPTransport) Get(ctx context.Context, key string) (*Value, error) {
	reqURL := fmt.Sprintf("%s/%s/%s", t.keeperURL, t.getSettingsEndpoint, key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "http-transport get key")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, NewBadStatusCodeErr(resp.StatusCode, reqURL)
	}

	value := new(Value)

	err = json.NewDecoder(resp.Body).Decode(value)
	if err != nil {
		return nil, errors.Wrap(err, "cannot decode keeper response")
	}

	return value, nil
}

func (t *HTTPTransport) GetAllSettings(ctx context.Context) ([]ExtendedValue, error) {
	reqURL := fmt.Sprintf("%s/%s", t.keeperURL, t.getAllSettingsEndpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "http-transport get all keys")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, NewBadStatusCodeErr(resp.StatusCode, reqURL)
	}

	var values []ExtendedValue

	err = json.NewDecoder(resp.Body).Decode(&values)
	if err != nil {
		return nil, errors.Wrap(err, "cannot decode keeper response")
	}

	return values, nil
}

func (t *HTTPTransport) GetAllLocalizationErrors(ctx context.Context) (map[string]map[string]string, error) {
	reqURL := t.keeperURL + "/admin/outside/mappingerrors/localization"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "http-transport get bank localizations")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, NewBadStatusCodeErr(resp.StatusCode, reqURL)
	}

	mappingLocalizationErrors := make(map[string]map[string]string)

	err = json.NewDecoder(resp.Body).Decode(&mappingLocalizationErrors)
	if err != nil {
		return nil, errors.Wrap(err, "cannot decode keeper response")
	}

	return mappingLocalizationErrors, nil
}

func (t *HTTPTransport) GetAllBankErrors(ctx context.Context, bank string) (map[string]string, error) {
	reqURL := t.keeperURL + "/admin/outside/mappingerrors/bank/" + bank

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "http-transport get bank error")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, NewBadStatusCodeErr(resp.StatusCode, reqURL)
	}

	mappingBankErrors := make(map[string]string)

	err = json.NewDecoder(resp.Body).Decode(&mappingBankErrors)
	if err != nil {
		return nil, errors.Wrap(err, "cannot decode keeper response")
	}

	return mappingBankErrors, nil
}

func NewHTTPTransport(
	baseURL,
	getSettingsEndpoint,
	getAllSettingsEndpoint,
	proxyURL string,
	timeout time.Duration,
) (*HTTPTransport, error) {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: time.Duration(2) * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: time.Duration(2) * time.Second,
	}

	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("can't parse proxy url: %w", err)
		}

		transport.Proxy = http.ProxyURL(proxy)
	}

	if getSettingsEndpoint == "" {
		getSettingsEndpoint = "admin/get"
	}

	if getAllSettingsEndpoint == "" {
		getAllSettingsEndpoint = "admin/get_all"
	}

	return &HTTPTransport{
		keeperURL:              baseURL,
		getSettingsEndpoint:    getSettingsEndpoint,
		getAllSettingsEndpoint: getAllSettingsEndpoint,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}, nil
}
