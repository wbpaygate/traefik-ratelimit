package traefik_ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"gitlab-private.wildberries.ru/wbpay-go/traefik-ratelimit/internal/keeper"
	"golang.org/x/time/rate"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//const xRequestIDHeader = "X-Request-Id"

func CreateConfig() *Config {
	return &Config{}
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) (err error) {
	d.Duration, err = time.ParseDuration(strings.Trim(string(b), `"`))
	return
}

func (d Duration) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

type Config struct {
	KeeperRateLimitKey  string   `json:"keeperRateLimitKey,omitempty"`
	KeeperURL           string   `json:"keeperURL,omitempty"`
	KeeperReqTimeout    Duration `json:"keeperReqTimeout,omitempty"`
	KeeperAdminPassword string   `json:"keeperAdminPassword,omitempty"`
}

type klimit struct {
	EndpointPat string `json:"endpointpat"`
	HeaderKey   string `json:"headerkey"`
	HeaderVal   string `json:"headerval"`
}

type climit struct {
	klimit
	Limit rate.Limit `json:"limit"`
}

type limit struct {
	klimit
	Limit   rate.Limit
	limiter *rate.Limiter
}

type limits3 struct {
	key    string
	limits map[string]*limit
}

type limits2 struct {
	limits []limits3
	limit  *limit
}

type limits struct {
	limits  map[string]*limits2
	mlimits map[klimit]*limit
	ipat    [][]int
}

type RateLimit struct {
	name     string
	next     http.Handler
	config   *Config
	version  *keeper.Resp
	settings Settings
	mtx      sync.Mutex
	limits   atomic.Pointer[limits]
}

type Settings interface {
	Get(key string) (*keeper.Resp, error)
}


// New created a new plugin.
func New(ctx context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	mlog(fmt.Sprintf("config %v", cfg))
	if len(cfg.KeeperRateLimitKey) == 0 {
		return nil, fmt.Errorf("config: keeperRateLimitKey is empty")
	}

	if len(cfg.KeeperURL) == 0 {
		return nil, fmt.Errorf("config: keeperURL is empty")
	}

	if len(cfg.KeeperAdminPassword) == 0 {
		return nil, fmt.Errorf("config: keeperAdminPassword is empty")
	}

	if cfg.KeeperReqTimeout.Duration == 0 {
		cfg.KeeperReqTimeout.Duration = 300 * time.Second
	}

	r := newRateLimit(next, cfg, name)
	err := r.setFromSettings()
	if err != nil {
		return nil, err
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err := r.setFromSettings()
				if err != nil {
					mlog("cant get ratelimits from keeper", err)
				}
			}
		}
	}()

	return r, nil
}

func newRateLimit(next http.Handler, cfg *Config, name string) *RateLimit {
	r := &RateLimit{
		name:     name,
		next:     next,
		config:   cfg,
		settings: keeper.New(cfg.KeeperURL, cfg.KeeperReqTimeout.Duration, cfg.KeeperAdminPassword),
	}
	r.limits.Store(&limits{
		limits:  make(map[string]*limits2),
		mlimits: make(map[klimit]*limit),
		ipat:    make([][]int, 0),
	})
	return r
}

func (r *RateLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	encoder := json.NewEncoder(rw)
	//	requestID := req.Header.Get(xRequestIDHeader)

//	reqCtx := req.Context()
	//	reqCtx = context.WithValue(reqCtx, "requestID", requestID)
	//	reqCtx = context.WithValue(reqCtx, "env", r.config.Env)

//	if r.Allow(reqCtx, req, rw) {
	if r.Allow(req) {
		r.next.ServeHTTP(rw, req)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusTooManyRequests)
	encoder.Encode(map[string]any{"status_code": http.StatusTooManyRequests, "message": "rate limit exceeded, try again later"})
	return
}

func (r *RateLimit) setFromSettings() error {
	result, err := r.settings.Get(r.config.KeeperRateLimitKey)
	if err != nil {
		return err
	}
	if result != nil && !r.version.Equal(result) {
		err = r.update([]byte(result.Value))
		if err != nil {
			return err
		}
		r.version = result
	}

	return nil
}

func mlog(args ...any) {
	os.Stdout.WriteString(fmt.Sprintf("[rate-limit-middleware-plugin] %s\n", fmt.Sprint(args...)))
}

func allow(lim map[string]*limits2, p string, req *http.Request) (bool, bool) {
	if ls2, ok := lim[p]; ok {
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

//func (r *RateLimit) Allow(ctx context.Context, req *http.Request, rw http.ResponseWriter) bool {
func (r *RateLimit) Allow(req *http.Request) bool {
	lim := r.limits.Load()
//	fmt.Println("lim.ipat", lim.ipat)
	for _, ipt := range lim.ipat {
//		fmt.Println("ipat", ipt)
		if p, ok := preparepat(ipt, req.URL.Path); ok {
//			fmt.Println("p", p, ok)
			if res, ok := allow(lim.limits, p, req); ok {
				return res
			}
		}
	}
	if res, ok := allow(lim.limits, "", req); ok {
		return res
	}
	return true
}

func appendipat(ipat [][]int, ipt []int) [][]int {
	if ipt == nil {
		return ipat
	}
	for _, tipt := range ipat {
		if slices.Equal(tipt, ipt) {
			return ipat
		}
	}
	return append(ipat, ipt)
}

func preparepat(ipt []int, s string) (string, bool) {
//	fmt.Println("prep", s)
	ss := strings.Split(s, "/")
	r := make([]string, 0, len(ipt))
	for _, i := range ipt {
		j := i
		if i < 0 {
			j = len(ss) + i
		}
		if j > len(ss)-1 || j < 0 {
			return "", false
		}
//		fmt.Println("prep", i, len(ss), j)
		r = append(r, strconv.Itoa(i)+":"+ss[j])
	}
	return strings.Join(r, "/"), true
}

func compilepat(s string) (string, []int, error) {
	if len(strings.TrimSpace(s)) == 0 {
		return "", nil, nil
	}
	f := 0
	fl := false
	ss := strings.Split(s, "/")
	r := make([]string, 0, len(ss))
	ri := make([]int, 0, len(ss))
	for i, s := range ss {
		switch s {
		case "**":
			fl = true
			if f > 0 {
				return "", nil, fmt.Errorf("bad pattern")
			}
			f = i + 1
		case "*", "":
		default:
			r = append(r, s)
			ri = append(ri, i)
		}
	}
	for i := range r {
		if ri[i] >= f && fl {
			ri[i] = ri[i] - ri[len(ri)-1] - 1
		}
		r[i] = strconv.Itoa(ri[i]) + ":" + r[i]
	}
	return strings.Join(r, "/"), ri, nil
}

func (r *RateLimit) update(b []byte) error {
	type conflimits struct {
		Limits []climit `json:"limits"`
	}

//	fmt.Println("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	var clim conflimits
	if err := json.Unmarshal(b, &clim); err != nil {
		return err
	}
	var k klimit
	ep2 := make(map[klimit]struct{}, len(clim.Limits))
	j := 0
	for i := 0; i < len(clim.Limits); i++ {
		if len(clim.Limits[i].HeaderKey) == 0 || len(clim.Limits[i].HeaderVal) == 0 {
			clim.Limits[i].HeaderKey = ""
			clim.Limits[i].HeaderVal = ""
		}
		if len(clim.Limits[i].EndpointPat) == 0 && len(clim.Limits[i].HeaderKey) == 0 && len(clim.Limits[i].HeaderVal) == 0 {
			continue
		}
		if len(clim.Limits[i].HeaderKey) != 0 {
			clim.Limits[i].HeaderKey = http.CanonicalHeaderKey(clim.Limits[i].HeaderKey)
		}
		if len(clim.Limits[i].HeaderVal) != 0 {
			clim.Limits[i].HeaderVal = strings.ToLower(clim.Limits[i].HeaderVal)
		}
		k = klimit{
			EndpointPat: clim.Limits[i].EndpointPat,
			HeaderKey:   clim.Limits[i].HeaderKey,
			HeaderVal:   clim.Limits[i].HeaderVal,
		}
		if _, ok := ep2[k]; ok {
			continue
		}
		ep2[k] = struct{}{}
		if j != i {
			clim.Limits[j].Limit = clim.Limits[i].Limit
		}
		j++
	}
	clim.Limits = clim.Limits[:j]

//	fmt.Println("limits", clim.Limits)

	r.mtx.Lock()
	defer r.mtx.Unlock()

	oldlim := r.limits.Load()
	if len(clim.Limits) == len(oldlim.mlimits) {
		ch := false
		for _, l := range clim.Limits {
			k = klimit{
				EndpointPat: l.EndpointPat,
				HeaderKey:   l.HeaderKey,
				HeaderVal:   l.HeaderVal,
			}
			if l2, ok := oldlim.mlimits[k]; ok {
				if l2.Limit == l.Limit {
					continue
				}
				l2.limiter.SetLimit(l.Limit)
				l2.Limit = l.Limit
			} else {
				ch = true
			}
		}
		if !ch {
			return nil
		}
	}

	newlim := limits{
		limits:  make(map[string]*limits2, len(clim.Limits)),
		mlimits: make(map[klimit]*limit, len(clim.Limits)),
		ipat:    make([][]int, 0, len(clim.Limits)),
	}
lim:
	for _, l := range clim.Limits {
		k = klimit{
			EndpointPat: l.EndpointPat,
			HeaderKey:   l.HeaderKey,
			HeaderVal:   l.HeaderVal,
		}
		lim := oldlim.mlimits[k]
		if lim == nil {
			lim = &limit{
				klimit:  k,
				Limit:   l.Limit,
				limiter: rate.NewLimiter(l.Limit, 1),
			}
		}
		newlim.mlimits[k] = lim
		p, ipt, err := compilepat(l.EndpointPat)
		if err != nil {
			return err
		}
		newlim.ipat = appendipat(newlim.ipat, ipt)
		lim2, ok := newlim.limits[p]
		if !ok {
			if len(l.HeaderKey) == 0 {
				newlim.limits[p] = &limits2{
					limit: lim,
				}
			} else {
				newlim.limits[p] = &limits2{
					limits: []limits3{
						limits3{
							key: l.HeaderKey,
							limits: map[string]*limit{
								l.HeaderVal: lim,
							},
						},
					},
				}
			}
			continue
		}
		if len(l.HeaderKey) == 0 {
			lim2.limit = lim
		} else {
			for i := 0; i < len(lim2.limits); i++ {
				if lim2.limits[i].key == l.HeaderKey {
					lim2.limits[i].limits[l.HeaderVal] = lim
					continue lim
				}
			}
			lim2.limits = append(lim2.limits, limits3{
				key: l.HeaderKey,
				limits: map[string]*limit{
					l.HeaderVal: lim,
				},
			})
		}
	}
	r.limits.Store(&newlim)

	return nil
}
