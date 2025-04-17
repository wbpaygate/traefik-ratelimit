package traefik_ratelimit

import (
	"net/http"
	"sync"

	"github.com/wbpaygate/traefik-ratelimit/internal/limiter"
	"github.com/wbpaygate/traefik-ratelimit/internal/logger"
)

func (rl *RateLimiter) Allow(req *http.Request) bool {
	rulesPtr := rl.rules.Load()
	rules, ok := rulesPtr.(*sync.Map)
	if !ok {
		logger.Error(req.Context(), "rules: cannot type assert *sync.Map")
		return true
	}

	var allow = true

	rules.Range(func(k, v any) bool {
		rule := k.(RuleImpl)
		lim := v.(*limiter.Limiter)
		if match := rule.URLPathPattern.Match([]byte(req.URL.Path)); match {
			if rule.Header != nil {
				if req.Header.Get(rule.Header.key) == rule.Header.val {
					allow = lim.Allow()
					return allow // это return из функции обхода мапы
				}
			}

			allow = lim.Allow()
		}

		return true
	})

	return allow
}
