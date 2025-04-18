package traefik_ratelimit

import (
	"bytes"
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
		return nil, fmt.Errorf("validate error: %w", err)
	}

	return &l, nil
}

func (rl *RateLimiter) loadLimits(limitsConfig []byte) error {
	l, err := serializeAndValidateLimits(limitsConfig)
	if err != nil {
		return fmt.Errorf("serializeAndValidateLimits error: %w", err)
	}

	settings, ok := rl.keeperSetting.Load().(*keeper.Value)
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
	if !ok {
		return fmt.Errorf("keeperClient not init, try reconfigure")
	}

	result, err := kc.GetRateLimits(ctx)
	if err != nil {
		return fmt.Errorf("failed to get limits from keeper, error: %w", err)
	}

	if result == nil || result.Value == "" {
		return fmt.Errorf("empty result from keeper")
	}

	logDebugJSON(ctx, result.Value)

	settings, ok := rl.keeperSetting.Load().(*keeper.Value)
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

		logger.Info(ctx, fmt.Sprintf("new configuration loaded: version: %d, mod_revision: %d", result.Version, result.ModRevision))

		rl.logWorkingLimits(ctx)

	} else {
		logger.Info(ctx, fmt.Sprintf("no update, use configuration: version: %d, mod_revision: %d", settings.Version, settings.ModRevision))
	}

	return nil
}

func (rl *RateLimiter) hotReloadLimits(limits *Limits) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	newRules := &sync.Map{}

	for _, limit := range limits.Limits {
		lim := limiter.NewLimiter(limit.Limit)

		for _, rule := range limit.Rules {
			ruleImpl := RuleImpl{
				URLPathPattern: pattern.NewPattern(rule.URLPathPattern),
			}

			if rule.HeaderKey != "" && rule.HeaderVal != "" {
				ruleImpl.Header = &Header{
					key: rule.HeaderKey,
					val: rule.HeaderVal,
				}
			}

			newRules.Store(ruleImpl, lim)
		}
	}

	// закрытие старых лимитеров
	if oldRules, ok := rl.rules.Load().(*sync.Map); ok {
		defer func() {
			oldRules.Range(func(key, value any) bool {
				if lim, ok := value.(*limiter.Limiter); ok {
					lim.Close()
				}

				return true
			})
		}()
	}

	rl.rules.Store(newRules) // атомарное переключение
}

func logDebugJSON(ctx context.Context, rawJSON string) {
	var compacted bytes.Buffer

	if err := json.Compact(&compacted, []byte(rawJSON)); err != nil {
		logger.Debug(ctx, "invalid JSON, logging raw:", rawJSON)
		return
	}

	logger.Debug(ctx, "raw limits from keeper:", compacted.String())
}
