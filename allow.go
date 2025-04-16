package traefik_ratelimit

import (
	"net/http"
	"sync"

	"github.com/wbpaygate/traefik-ratelimit/internal/limiter"
	"github.com/wbpaygate/traefik-ratelimit/internal/logger"
	"github.com/wbpaygate/traefik-ratelimit/internal/pattern"
)

func (rl *RateLimiter) Allow(req *http.Request) bool {
	return rl.allowFromHeaders(req) && rl.allowFromPatterns(req)
}

func (rl *RateLimiter) allowFromHeaders(req *http.Request) bool {
	headersPtr := rl.headers.Load()
	if headersPtr == nil {
		logger.Error(req.Context(), "headersPtr is nil")
		return true
	}

	headers, ok := headersPtr.(*sync.Map)
	if !ok {
		logger.Error(req.Context(), "cannot type assert *sync.Map")
		return true
	}

	var allow = true

	headers.Range(func(key, value any) bool {
		header := key.(*Header)
		lim := value.(*limiter.Limiter)

		if req.Header.Get(header.key) == header.val {
			allow = lim.Allow()
			return allow // это return из функции обхода мапы
		}

		return true // too
	})

	return allow
}

func (rl *RateLimiter) allowFromPatterns(req *http.Request) bool {
	patternsPtr := rl.patterns.Load()
	if patternsPtr == nil {
		logger.Error(req.Context(), "patternsPtr is nil")
		return true
	}

	patterns, ok := patternsPtr.(*sync.Map)
	if !ok {
		logger.Error(req.Context(), "cannot type assert *sync.Map")
		return true
	}

	allow := true

	patterns.Range(func(key, value any) bool {
		ptrn := key.(*pattern.Pattern)
		lim := value.(*limiter.Limiter)

		if ptrn.Match([]byte(req.URL.Path)) {
			allow = lim.Allow()
			return allow // это return из функции обхода мапы
		}

		return true // too
	})

	return allow
}
