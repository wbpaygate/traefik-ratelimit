package traefik_ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/wbpaygate/traefik-ratelimit/internal/keeper"
	"github.com/wbpaygate/traefik-ratelimit/internal/limiter"
	"github.com/wbpaygate/traefik-ratelimit/internal/logger"
	"github.com/wbpaygate/traefik-ratelimit/internal/pattern"
)

func serializeAndValidateLimits(b []byte) (*Limits, error) {
	var l Limits
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	if err := l.validate(); err != nil {
		return nil, fmt.Errorf("validate limit error: %w", err)
	}

	return &l, nil
}

func (rl *RateLimiter) loadLimits(limitsConfig []byte) error {
	l, err := serializeAndValidateLimits(limitsConfig)
	if err != nil {
		return fmt.Errorf("serializeAndValidateLimits error: %w", err)
	}

	settingsAny := rl.keeperSetting.Load()
	settings, ok := settingsAny.(*keeper.Value)
	if !ok {
		return fmt.Errorf("cannot type assert *keeper.Value")
	}
	if settings == nil {
		return fmt.Errorf("settings is nil")
	}

	settings.Version = 0
	settings.ModRevision = 0

	rl.limits.Store(l)
	rl.hotReloadLimits(l)

	return nil
}

func (rl *RateLimiter) updateLimits(ctx context.Context) error {
	kc, ok := rl.keeperClient.Load().(*keeper.KeeperClient)
	if !ok || kc == nil {
		return fmt.Errorf("keeperClient not init, try reconfigure")

	}

	result, err := kc.GetRateLimits(ctx)
	if err != nil {
		return fmt.Errorf("failed to get limits from keeper, error: %w", err)
	}

	if result == nil || result.Value == "" {
		return fmt.Errorf("empty result from keeper")
	}

	settingsAny := rl.keeperSetting.Load()
	settings, ok := settingsAny.(*keeper.Value)
	if !ok {
		return fmt.Errorf("cannot type assert *keeper.Value")
	}
	if settings == nil {
		return fmt.Errorf("settings is nil")
	}

	if !settings.Equal(result) {
		logger.Debug(ctx, fmt.Sprintf("old configuration: version: %d, mod_revision: %d", settings.Version, settings.ModRevision))

		l, err := serializeAndValidateLimits([]byte(result.Value))
		if err != nil {
			return fmt.Errorf("failed serialize and validate limits: %w", err)
		}

		rl.keeperSetting.Store(result)
		rl.limits.Store(l)

		rl.hotReloadLimits(l)

		logger.Debug(ctx, fmt.Sprintf("new configuration loaded: version: %d, mod_revision: %d", result.Version, result.ModRevision))

	} else {
		logger.Debug(ctx, fmt.Sprintf("no update, use configuration: version: %d, mod_revision: %d", settings.Version, settings.ModRevision))
	}

	return nil
}

func (rl *RateLimiter) hotReloadLimits(limits *Limits) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	newPatterns := &sync.Map{}
	newHeaders := &sync.Map{}

	for _, limit := range limits.Limits {
		lim := limiter.NewLimiter(limit.Limit)

		for _, rule := range limit.Rules {
			stored := false

			if rule.UrlPathPattern != "" {
				newPatterns.Store(pattern.NewPattern(rule.UrlPathPattern), lim)
				stored = true
			}

			if rule.HeaderKey != "" && rule.HeaderVal != "" {
				newHeaders.Store(&Header{
					key: rule.HeaderKey,
					val: rule.HeaderVal,
				}, lim)
				stored = true
			}

			if !stored {
				lim.Close()
			}
		}
	}

	// закрытие старых лимитеров
	if oldPatternsPtr := rl.patterns.Load(); oldPatternsPtr != nil {
		if oldPatterns, okTypeAssert := oldPatternsPtr.(*sync.Map); okTypeAssert {
			defer func() {
				oldPatterns.Range(func(key, value any) bool {
					if lim, ok := value.(*limiter.Limiter); ok {
						lim.Close()
					}
					return true
				})
			}()
		}
	}

	if oldHeadersPtr := rl.headers.Load(); oldHeadersPtr != nil {
		if oldHeaders, okTypeAssert := oldHeadersPtr.(*sync.Map); okTypeAssert {
			defer func() {
				oldHeaders.Range(func(key, value any) bool {
					if lim, ok := value.(*limiter.Limiter); ok {
						lim.Close()
					}
					return true
				})
			}()
		}
	}

	// атомарное переключение
	rl.patterns.Store(newPatterns)
	rl.headers.Store(newHeaders)
}
