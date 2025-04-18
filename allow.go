package traefik_ratelimit

import (
	"net/http"
	"sync"

	"github.com/wbpaygate/traefik-ratelimit/internal/limiter"
	"github.com/wbpaygate/traefik-ratelimit/internal/logger"
)

func (rl *RateLimiter) Allow(req *http.Request) bool {
	rules, ok := rl.rules.Load().(*sync.Map)
	if !ok {
		logger.Error(req.Context(), "rules: cannot type assert *sync.Map")
		return true
	}

	var allow = true

	rules.Range(func(k, v any) bool {
		if rule, okRule := k.(RuleImpl); okRule {
			if lim, okLim := v.(*limiter.Limiter); okLim {
				if !rule.URLPathPattern.Match([]byte(req.URL.Path)) {
					return true // это return из функции обхода мапы
				}

				if rule.Header != nil {
					if req.Header.Get(rule.Header.key) != rule.Header.val {
						return true // это return из функции обхода мапы
					}
				}

				allow = lim.Allow()
				return allow // это return из функции обхода мапы
			}
		}

		return true
	})

	return allow
}
