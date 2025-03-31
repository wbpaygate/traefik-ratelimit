package keeperclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ReneKroon/ttlcache"
	"github.com/pkg/errors"

	"gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/campaign"
	"gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/common"
	"gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/facilitator"
	"gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/routing"
	"gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/transport"
)

const (
	cacheKeyPrefix      = "keeper.value."
	cacheKeyAllSettings = "keeper.value.all_settings"

	defaultCacheTTL = 10 * time.Second
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, payload []byte, ttl time.Duration) error
}

type Logger interface {
	Error(ctx context.Context, err error)
	Info(ctx context.Context, message string)
}

type Option func(c *Client)

var WithLogger = func(logger Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

var WithColdCache = func(cache Cache, cacheKeyPrefix string, reloadInterval time.Duration) Option {
	return func(c *Client) { //nolint:varnamelen
		c.coldCache = cache
		c.cacheKeyPrefix = cacheKeyPrefix

		if reloadInterval < defaultCacheTTL {
			reloadInterval = defaultCacheTTL
		}

		c.cacheReloadInterval = reloadInterval
	}
}

var WithPreloadCache = func() Option {
	return func(c *Client) {
		c.withPreloadCache = true
	}
}

// Client структура для взаимодействия с сервисом keeper.
type Client struct {
	transport           transport.Transport
	cache               *ttlcache.Cache
	persistentCache     *ttlcache.Cache
	coldCache           Cache
	locks               locks
	logger              Logger
	cacheTTL            time.Duration
	cacheReloadInterval time.Duration
	cacheKeyPrefix      string
	withPreloadCache    bool
	stop                chan struct{}
}

func (c *Client) GetConfig(ctx context.Context, key string) (*campaign.Config, error) {
	resp, err := c.Get(ctx, key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get config")
	}

	cfg := new(campaign.Config)
	if err = json.Unmarshal([]byte(resp.Value), cfg); err != nil {
		return nil, errors.Wrap(err, "GetConfig: response unmarshaller")
	}

	cfg.WhiteListMap = common.ListToMap(cfg.WhiteList)
	cfg.BlackListMap = common.ListToMap(cfg.BlackList)

	return cfg, nil
}

func (c *Client) GetMap(ctx context.Context, key string) (map[string]string, error) {
	value, err := c.Get(ctx, key)
	if err != nil {
		return nil, errors.Wrap(err, "failed get map")
	}

	m := make(map[string]string)
	if err = json.Unmarshal([]byte(value.Value), &m); err != nil {
		return nil, errors.Wrap(err, "GetMap: response unmarshaller")
	}

	return m, nil
}

// Get возвращает текущее значение параметра в сервисе keeper.
// Нужно учитывать, что значения на время кешируются и обновляются с заданной периодичностью.
func (c *Client) Get(ctx context.Context, key string) (*transport.Value, error) {
	var (
		err   error
		value *transport.Value
	)

	if c.cache == nil {
		return nil, errors.New("cache is not inited")
	}

	if c.persistentCache == nil {
		return nil, errors.New("persistent cache is not inited")
	}

	cacheKey := cacheKeyPrefix + key

	if cachedValue, ok := c.cache.Get(cacheKey); ok {
		if value, ok = cachedValue.(*transport.Value); ok {
			return value, nil
		}
	}

	// Если стоит блокировка, значит кто-то уже обновляет кеш. В этом случае
	// пытаемся отдать предыдущее значение.
	if c.locks.Get(key) {
		if cachedValue, ok := c.persistentCache.Get(cacheKey); ok {
			if value, ok = cachedValue.(*transport.Value); ok {
				return value, nil
			}
		}

		return nil, NewPersistentCacheErr(key)
	}

	// Значение не найдено. Первый из запросов блокирует за собой обновление (на самом деле
	// может возникнуть ситуация когда несколько запросов поставят блокировку и начнут
	// обновлять кеш - пока считаем это некритичным).
	c.locks.Set(key, true)
	defer c.locks.Set(key, false)

	value, err = c.transport.Get(ctx, key)
	if err == nil {
		c.cache.SetWithTTL(cacheKey, value, c.cacheTTL)
		c.persistentCache.Set(cacheKey, value)

		return value, nil
	}

	c.logError(ctx, errors.Wrap(err, "could not get value from transport"))

	// Если не смогли сходить по сети, пробуем отдать значение из кеша
	if cachedValue, ok := c.persistentCache.Get(cacheKey); ok {
		if value, ok = cachedValue.(*transport.Value); ok {
			return value, nil
		}
	}

	return nil, errors.Wrap(err, "could not get value from transport")
}

// GetAllSettings возвращает все параметры в сервисе keeper.
// Нужно учитывать, что значения на время кешируются и обновляются с заданной периодичностью.
func (c *Client) GetAllSettings(ctx context.Context) (transport.ValuesStore, error) {
	values := make(transport.ValuesStore)

	if cachedValue, ok := c.cache.Get(cacheKeyAllSettings); ok {
		if values, ok = cachedValue.(transport.ValuesStore); ok {
			return values, nil
		}
	}

	if c.locks.Get(cacheKeyAllSettings) {
		if cachedValue, ok := c.persistentCache.Get(cacheKeyAllSettings); ok {
			if values, ok = cachedValue.(transport.ValuesStore); ok {
				return values, nil
			}
		}

		return nil, NewPersistentCacheErr(cacheKeyAllSettings)
	}

	c.locks.Set(cacheKeyAllSettings, true)
	defer c.locks.Set(cacheKeyAllSettings, false)

	records, err := c.transport.GetAllSettings(ctx)
	if err == nil {
		for _, record := range records {
			values[record.Key] = record
		}

		c.cache.SetWithTTL(cacheKeyAllSettings, values, c.cacheTTL)
		c.persistentCache.Set(cacheKeyAllSettings, values)

		return values, nil
	}

	c.logError(ctx, errors.Wrap(err, "could not get all values from transport"))

	// Если не смогли сходить по сети, пробуем отдать значение из кеша
	if cachedValue, ok := c.persistentCache.Get(cacheKeyAllSettings); ok {
		if values, ok = cachedValue.(transport.ValuesStore); ok {
			return values, nil
		}
	}

	return nil, errors.Wrap(err, "could not get values from transport")
}

// GetFallback возвращает значение параметра из сервиса keeper в случае успеха и
// fallback-значение в случае ошибки. Обработав второе значение в возвращаемых параметрах,
// можно определить вернулось ли значение по умолчанию.
func (c *Client) GetFallback(ctx context.Context, key, fallbackValue string) (*transport.Value, bool) {
	var isFallback bool

	value, err := c.Get(ctx, key)
	if err != nil {
		isFallback = true
		value = &transport.Value{Value: fallbackValue}
	}

	return value, isFallback
}

func getAndUnmarshal[T any](ctx context.Context, keeper KeeperClient, key string) (val T, err error) {
	response, err := keeper.Get(ctx, key)
	if err != nil {
		return val, fmt.Errorf("keeperClient: %w", err)
	}

	if err := json.Unmarshal([]byte(response.Value), &val); err != nil {
		return val, fmt.Errorf("json.Unmarshal: %w, %v", err, []byte(response.Value))
	}

	return val, nil
}

func GetFunc[T any](keeper KeeperClient) func(ctx context.Context, key string) (T, error) {
	return func(ctx context.Context, key string) (val T, err error) {
		return getAndUnmarshal[T](ctx, keeper, key)
	}
}

func GetKeyFunc[T any](keeper KeeperClient, key string) func(ctx context.Context) (T, error) {
	return func(ctx context.Context) (val T, err error) {
		return getAndUnmarshal[T](ctx, keeper, key)
	}
}

func (c *Client) IsFeatureActiveBank(ctx context.Context, key string, dto routing.FeatureBankDto) (bool, error) {
	activeForUser, err := c.IsFeatureActive(ctx, key, dto.UserID)
	if err != nil || !activeForUser {
		return false, err
	}

	bank := routing.Bank(dto.Bank)
	resp, err := c.Get(ctx, key)

	if err != nil {
		return false, err
	}

	cfg := new(routing.Config)
	err = json.Unmarshal([]byte(resp.Value), cfg)

	if err != nil {
		return false, errors.Wrap(err, "IsFeatureActiveBank: response unmarshaller")
	}

	if len(dto.Bank) == 0 {
		return false, nil
	}

	return bank.IsIncludedInCampaign(cfg), nil
}

func (c *Client) IsFeatureActiveRouting(ctx context.Context, key string, dto routing.FeatureRoutingDto) (bool, error) {
	activeForUser, err := c.IsFeatureActive(ctx, key, dto.UserID)
	if err != nil || !activeForUser {
		return false, err
	}

	member := routing.Member(dto.Member)
	resp, err := c.Get(ctx, key)

	if err != nil {
		return false, err
	}

	cfg := new(routing.Config)
	err = json.Unmarshal([]byte(resp.Value), cfg)

	if err != nil {
		return false, errors.Wrap(err, "IsFeatureActiveRouting: response unmarshaller")
	}

	if len(dto.Member) == 0 {
		return false, nil
	}

	return member.IsIncludedInCampaign(cfg), nil
}

// IsFeatureActive возвращает признак вовлеченности пользователя в какой-либо функционал.
func (c *Client) IsFeatureActive(ctx context.Context, key string, user string) (bool, error) {
	cfg, err := c.GetConfig(ctx, key)
	if err != nil {
		return false, err
	}

	if !cfg.Enabled {
		return false, nil
	}

	return userID(user).IsIncludedInCampaign(cfg), nil
}

func (c *Client) IsInFacilitatorScheme(ctx context.Context, key, user, bankName, cardIssuer, paymentSystem string) (bool, error) {
	resp, err := c.Get(ctx, key)
	if err != nil {
		return false, err
	}

	scheme := make(map[string]facilitator.FsScheme)

	err = json.Unmarshal([]byte(resp.Value), &scheme)
	if err != nil {
		return false, errors.Wrap(err, "IsFeatureActive: response unmarshaller")
	}

	cfg, findedInScheme := scheme[bankName]
	if !findedInScheme {
		return false, nil
	}

	cfg.WhiteListMap = common.ListToMap(cfg.WhiteList)
	cfg.BlackListMap = common.ListToMap(cfg.BlackList)

	if !cfg.Enabled {
		return false, nil
	}

	if !userID(user).IsIncludedInCampaign(&cfg.Config) {
		return false, nil
	}

	if cardIssuer == cfg.Self && cfg.Onus.PaymentSystem != nil {
		v, ok := cfg.Onus.PaymentSystem[paymentSystem]
		return ok && v, nil
	}

	v, ok := cfg.Offus.PaymentSystem[paymentSystem]
	if ok && v {
		return true, nil
	}

	return false, nil
}

// IsFeatureActiveFallback возвращает признак вовлеченности пользователя в
// какой-либо функционал. В отличие от IsFeatureActive позволяет указать значение
// по умолчанию, в случае возникновения ошибки.
func (c *Client) IsFeatureActiveFallback(ctx context.Context, key, user string, fallbackVal bool) (bool, bool) {
	var isFallback bool

	res, err := c.IsFeatureActive(ctx, key, user)
	if err != nil {
		isFallback = true
		res = fallbackVal
	}

	return res, isFallback
}

func New(cfg Config, opts ...Option) (*Client, error) {
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = defaultCacheTTL
	}

	cache := ttlcache.NewCache()
	cache.SkipTtlExtensionOnHit(true)

	httpTransport, err := transport.NewHTTPTransport(
		cfg.KeeperURL,
		cfg.KeeperSettingsPath,
		cfg.KeeperSettingsAllPath,
		cfg.KeeperProxy,
		cfg.ReqTimeout,
	)
	if err != nil {
		return nil, fmt.Errorf("can't create http transport: %w", err)
	}

	cli := &Client{
		transport:       httpTransport,
		cache:           cache,
		persistentCache: ttlcache.NewCache(),
		locks:           locks{keys: map[string]bool{}},
		cacheTTL:        cfg.CacheTTL,
		stop:            make(chan struct{}),
	}

	for _, opt := range opts {
		opt(cli)
	}

	return cli, nil
}

func (c *Client) Start(ctx context.Context) error {
	if c.withPreloadCache {
		if err := c.preloadCache(ctx); err != nil {
			return errors.Wrap(err, "preload cache failed")
		}
	}

	if c.coldCache != nil {
		go c.reloadColdCache(ctx)
	}

	return nil
}

func (c *Client) Stop() {
	close(c.stop)
}
