package keeperclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/transport"
)

func (c *Client) preloadCache(ctx context.Context) error {
	records, err := c.GetAllSettings(ctx)
	if err != nil {
		c.logError(ctx, errors.Wrap(err, "could not get all settings from keeper"))
		if c.coldCache == nil { //nolint:wsl
			return errors.Wrap(err, "could not get values from keeper")
		}
		// Если не смогли забрать параметры по сети, пробуем забрать из холодного кеша
		records, err = c.getColdCache(ctx)
		if err != nil {
			return errors.Wrap(err, "could not load cold cache")
		}
	}

	for _, record := range records {
		value := &transport.Value{
			Value:       record.Value,
			ModRevision: record.ModRevision,
			Version:     record.Version,
		}
		c.cache.SetWithTTL(cacheKeyPrefix+record.Key, value, c.cacheTTL)
		c.persistentCache.Set(cacheKeyPrefix+record.Key, value)
	}

	c.logInfo(ctx, fmt.Sprintf("preload cache. records count: %d", len(records)))

	if c.coldCache != nil {
		if err = c.setColdCache(ctx, records); err != nil {
			return errors.Wrap(err, "could not set cold cache")
		}

		c.logInfo(ctx, fmt.Sprintf("cold cache initialized. records count: %d", len(records)))
	}

	return nil
}

func (c *Client) reloadColdCache(ctx context.Context) {
	if c.coldCache == nil {
		return
	}

	ticker := time.NewTicker(c.cacheReloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			records, err := c.transport.GetAllSettings(ctx)
			if err != nil {
				c.logError(ctx, errors.Wrap(err, "could not get all settings from keeper"))
				continue
			}

			values := make(transport.ValuesStore)
			for _, record := range records {
				values[record.Key] = record
			}

			if err = c.setColdCache(ctx, values); err != nil {
				c.logError(ctx, errors.Wrap(err, "could not set cold cache"))
			}
		case <-c.stop:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) getColdCache(ctx context.Context) (transport.ValuesStore, error) {
	if c.coldCache == nil {
		return nil, errors.New("cold cache is not inited")
	}

	cachedValue, err := c.coldCache.Get(ctx, fmt.Sprintf("%s.%s", c.cacheKeyPrefix, cacheKeyAllSettings))
	if err != nil {
		return nil, errors.Wrap(err, "could not get value from cold cache")
	}

	var values transport.ValuesStore
	if err = json.Unmarshal(cachedValue, &values); err != nil {
		return nil, errors.Wrap(err, "could not unmarshal values from cold cache")
	}

	return values, nil
}

func (c *Client) setColdCache(ctx context.Context, records transport.ValuesStore) error {
	if c.coldCache == nil {
		return errors.New("cold cache is not inited")
	}

	val, err := json.Marshal(records)
	if err != nil {
		return errors.Wrap(err, "could not marshal records")
	}

	if err = c.coldCache.Set(
		ctx,
		fmt.Sprintf("%s.%s", c.cacheKeyPrefix, cacheKeyAllSettings),
		val,
		0,
	); err != nil {
		return errors.Wrap(err, "could not set value in cold cache")
	}

	return nil
}

func (c *Client) logError(ctx context.Context, err error) {
	if c.logger != nil {
		c.logger.Error(ctx, err)
	}
}

func (c *Client) logInfo(ctx context.Context, msg string) {
	if c.logger != nil {
		c.logger.Info(ctx, msg)
	}
}
