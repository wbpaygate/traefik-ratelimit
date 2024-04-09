package traefik_ratelimit

import (
	"github.com/wbpaygate/traefik-ratelimit/internal/pat2"
	"net/http"
	"strings"
	"sync/atomic"
)

func (r *RateLimit) allow1(p string, req *http.Request) (bool, bool) {
	if ls2, ok := grl.limits.limits[p]; ok {
		for _, ls3 := range ls2.limits {
			val := strings.ToLower(req.Header.Get(ls3.key))
			if len(val) == 0 {
				continue
			}
			if l, ok := ls3.limits[val]; ok {
				return l.limiter.Allow(), true
			}
		}
		if ls2.limit != nil {
			return ls2.limit.limiter.Allow(), true
		}
	}
	return false, false
}

func (r *RateLimit) Allow(req *http.Request) bool {
	return r.allow(req)
}

func (r *RateLimit) allow(req *http.Request) bool {
	cnt := atomic.AddInt32(r.cnt, 1)
	if cnt%1000 == 0 {
		r.log("allow ", cnt)
	}
	grl.mtx.RLock()
	defer grl.mtx.RUnlock()
	for _, ipt := range grl.limits.pats {
		if p, ok := pat.Preparepat(ipt, req.URL.Path); ok {
			if res, ok := r.allow1(p, req); ok {
				return res
			}
		}
	}
	if res, ok := r.allow1("", req); ok {
		return res
	}
	return true
}
