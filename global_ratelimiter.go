package traefik_ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wbpaygate/traefik-ratelimit/internal/keeper"
	"github.com/wbpaygate/traefik-ratelimit/internal/limiter"
	"github.com/wbpaygate/traefik-ratelimit/internal/logger"
	"github.com/wbpaygate/traefik-ratelimit/internal/pattern"
)

const (
	defaultTickerPeriod        = 30 * time.Second
	defaultKeeperClientTimeout = 3 * time.Second

	defaultRateLimitLimits = `{"limits": []}`
)

var globalRateLimiter *RateLimiter

// init нужен для того, чтобы разделить RateLimiter и TraefikRateLimiter
// т.к. RateLimiter не может иметь методов кроме ServeHTTP
func init() {
	cfg := &Config{
		KeeperURL:              "",
		KeeperSettingsEndpoint: "",
		KeeperRateLimitKey:     "ratelimits",
		KeeperReqTimeout:       "15s",
		KeeperReloadInterval:   "30s",
		RatelimitDebug:         "false",
		RatelimitData:          defaultRateLimitLimits,
	}

	ctx := context.Background()

	globalRateLimiter = NewRateLimiter(ctx, defaultRateLimitLimits)
	globalRateLimiter.Configure(ctx, cfg, nil)

	logger.Info(ctx, "init global rate limiter")
}

type RateLimiter struct {
	limits        atomic.Value // *Limits
	keeperSetting atomic.Value // *keeper.Value

	patterns atomic.Value // *sync.Map
	headers  atomic.Value // *sync.Map
	mu       sync.Mutex   // нужен для релоада

	keeperClient atomic.Value
	ticker       atomic.Value
}

func NewRateLimiter(ctx context.Context, rateLimitLimits string) *RateLimiter {
	rl := &RateLimiter{
		limits:        atomic.Value{},
		keeperSetting: atomic.Value{},

		patterns: atomic.Value{},
		headers:  atomic.Value{},

		keeperClient: atomic.Value{},
		ticker:       atomic.Value{},
	}

	rl.keeperSetting.Store(&keeper.Value{
		Value:       rateLimitLimits,
		Version:     0,
		ModRevision: 0,
	})

	rl.patterns.Store(&sync.Map{})
	rl.headers.Store(&sync.Map{})

	rl.keeperClient.Store((*keeper.KeeperClient)(nil))

	rl.ticker.Store(&time.Ticker{})

	rl.logWorkingLimits(ctx)

	return rl
}

func (rl *RateLimiter) startBackgroundLimitsUpdater(ctx context.Context, tickerPeriod time.Duration) {
	if tickerPeriod < 2 {
		tickerPeriod = 2 // защита на уровне кода
	}

	ticker := time.NewTicker(tickerPeriod)
	rl.ticker.Store(ticker)

	go func() {
		for {
			select {
			case <-ticker.C:
				tickerCtx, cancel := context.WithTimeout(ctx, tickerPeriod-1)
				logger.Debug(tickerCtx, "try update limits")
				if err := rl.updateLimits(tickerCtx); err != nil {
					logger.Error(tickerCtx, fmt.Sprintf("cannot update limits, error: %v", err))
				}

				cancel()
			}
		}
	}()
}

func (rl *RateLimiter) Configure(ctx context.Context, cfg *Config, kc *keeper.KeeperClient) {
	if ctx == nil {
		ctx = context.Background()
	}

	if cfg.RatelimitData == "" {
		cfg.RatelimitData = defaultRateLimitLimits
	}

	keeperClientTimeout := defaultKeeperClientTimeout
	if du, err := time.ParseDuration(cfg.KeeperReqTimeout); err == nil {
		keeperClientTimeout = du
	}

	if kc == nil {
		cl := &http.Client{
			Timeout: keeperClientTimeout,
		}

		kc = keeper.NewKeeperClient(cl, cfg.KeeperURL, cfg.KeeperSettingsEndpoint, cfg.KeeperRateLimitKey)
	}

	rl.keeperClient.Store(kc)

	tickerPeriod := defaultTickerPeriod
	if du, err := time.ParseDuration(cfg.KeeperReloadInterval); err == nil {
		tickerPeriod = du
	}

	if err := rl.loadLimits([]byte(cfg.RatelimitData)); err != nil {
		logger.Error(ctx, fmt.Sprintf("cannot load limits from config, error: %v", err))

	} else {
		logger.Info(ctx, "update limits from config")
	}

	oldTickerPtr := rl.ticker.Load()
	if oldTickerPtr != nil {
		if oldTicker, ok := oldTickerPtr.(*time.Ticker); ok {
			if oldTicker != nil {
				oldTicker.Stop()
				select {
				case <-oldTicker.C: // ждём когда старый тикер остановится
				default:
				}
			}
		}
	}

	rl.startBackgroundLimitsUpdater(ctx, tickerPeriod)

	logger.Debug(ctx, "configure global rate limiter")
	rl.logWorkingLimits(ctx)
}

func (rl *RateLimiter) logWorkingLimits(ctx context.Context) {
	var patternsData []string
	var headersData []string

	if patternsPtr := rl.patterns.Load(); patternsPtr != nil {
		if patterns, okTypeAssert := patternsPtr.(*sync.Map); okTypeAssert {
			patterns.Range(func(key, value any) bool {
				if ptrn, ok := key.(*pattern.Pattern); ok {
					if lim, ok := value.(*limiter.Limiter); ok {
						patternsData = append(patternsData, ptrn.String()+", limit:"+strconv.Itoa(lim.Limit()))
					}
				}
				return true
			})
		}

	} else {
		logger.Error(ctx, "patterns is nil")
	}

	if headersPtr := rl.headers.Load(); headersPtr != nil {
		if headers, okTypeAssert := headersPtr.(*sync.Map); okTypeAssert {
			headers.Range(func(key, value any) bool {
				if header, ok := key.(*Header); ok {
					if lim, ok := value.(*limiter.Limiter); ok {
						patternsData = append(patternsData, header.String()+", limit:"+strconv.Itoa(lim.Limit()))
					}
				}
				return true
			})
		}

	} else {
		logger.Error(ctx, "headers is nil")
	}

	logger.Info(ctx, "current rate limits overview", "patterns", patternsData, "headers", headersData)
}
