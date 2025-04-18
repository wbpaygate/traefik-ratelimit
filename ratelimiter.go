package traefik_ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/wbpaygate/traefik-ratelimit/internal/logger"
)

// Config структура которую будет создавать traefik
// с помощью конструктора CreateConfig
//
// имя ключа как и тег должны быть одинаковые,
// за исключением первого символа
// он должен быть в "апперкейсе", а тег в "ловеркейсе"
type Config struct {
	KeeperURL              string `json:"keeperURL,omitempty"`
	KeeperSettingsEndpoint string `json:"keeperSettingsEndpoint,omitempty"`
	KeeperRateLimitKey     string `json:"keeperRateLimitKey,omitempty"`
	KeeperReqTimeout       string `json:"keeperReqTimeout,omitempty"`
	KeeperReloadInterval   string `json:"keeperReloadInterval,omitempty"`
	RatelimitDebug         string `json:"ratelimitDebug,omitempty"`
	RatelimitData          string `json:"ratelimitData,omitempty"`
}

func CreateConfig() *Config {
	return &Config{
		RatelimitData: defaultRateLimitLimits,
	}
}

// TraefikRateLimiter эта структура существует только по той причине
// что traefik со своим yaegi не может принять,
// что у структуры которую возвращает конструктор New()
// могут существовать какие либо методы кроме ServeHTTP (публичные и приватные)
type TraefikRateLimiter struct {
	next http.Handler
}

func (rl *TraefikRateLimiter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	encoder := json.NewEncoder(rw)

	if globalRateLimiter.Allow(req) {
		rl.next.ServeHTTP(rw, req)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusTooManyRequests)
	_ = encoder.Encode(map[string]any{"error_code": "ERR_TOO_MANY_REQUESTS", "error_description": "Слишком много запросов. Повторите попытку позднее."})

}

// New created a new plugin.
func New(ctx context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	logConfig(ctx, cfg)

	debug, _ := strconv.ParseBool(cfg.RatelimitDebug)
	logger.SetDebug(ctx, debug)

	globalRateLimiter.Configure(ctx, cfg, nil)

	logger.Debug(ctx, "new rate limiter")

	return &TraefikRateLimiter{
		next: next,
	}, nil
}

func logConfig(ctx context.Context, cfg *Config) {
	configJSON, err := json.Marshal(cfg)
	if err != nil {
		logger.Error(ctx, fmt.Sprintf("failed to marshal config: %v", err))
		return
	}

	logger.Info(ctx, "loaded config: "+string(configJSON))
}
