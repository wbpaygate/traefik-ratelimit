package traefik_ratelimit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wbpaygate/traefik-ratelimit/internal/keeper"
	"github.com/wbpaygate/traefik-ratelimit/internal/pat2"
	"github.com/wbpaygate/traefik-ratelimit/internal/rate"
)

const DEBUG = false

func CreateConfig() *Config {
	return &Config{
		KeeperReloadInterval: "30s",
		RatelimitData:        `{"limits": []}`,
	}
}

type Config struct {
	KeeperRateLimitKey   string `json:"keeperRateLimitKey,omitempty"`
	KeeperURL            string `json:"keeperURL,omitempty"`
	KeeperReqTimeout     string `json:"keeperReqTimeout,omitempty"`
	KeeperAdminPassword  string `json:"keeperAdminPassword,omitempty"`
	RatelimitPath        string `json:"ratelimitPath,omitempty"`
	RatelimitData        string `json:"ratelimitData,omitempty"`
	KeeperReloadInterval string `json:"keeperReloadInterval,omitempty"`
}

type rule struct {
	UrlPathPattern string `json:"urlpathpattern"`
	HeaderKey      string `json:"headerkey"`
	HeaderVal      string `json:"headerval"`
}

type limit struct {
	Limit   int
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
	mlimits map[rule]*limit
	pats    [][]pat.Pat
}

type RateLimit struct {
	name string
	next http.Handler
	cnt  *int32
	l    *log.Logger
}

type GlobalRateLimit struct {
	config    *Config
	version   *keeper.Resp
	settings  keeper.Settings
	umtx      sync.Mutex
	curlimit  *int32
	limits    []*limits
	rawlimits []byte
	ticker    *time.Ticker
	tickerto  time.Duration
	icnt      *int32
}

var grl *GlobalRateLimit

const LIMITS = 5

func init() {
	grl = &GlobalRateLimit{
		curlimit:  new(int32),
		limits:    make([]*limits, LIMITS),
		version:   &keeper.Resp{},
		rawlimits: []byte(""),
		icnt:      new(int32),
	}
	grl.limits[0] = &limits{
		limits:  make(map[string]*limits2),
		mlimits: make(map[rule]*limit),
		pats:    make([][]pat.Pat, 0),
	}

	config := CreateConfig()
	to := 30 * time.Second
	if du, err := time.ParseDuration(string(config.KeeperReloadInterval)); err == nil {
		to = du
	}
	grl.ticker = time.NewTicker(to)
	grl.tickerto = to
	grl.configure(nil, config)
	go func() {
		for {
			select {
			case <-grl.ticker.C:
				grl.sync()
			}
		}
	}()
	locallog("init")
}

// New created a new plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	var l *log.Logger
	if DEBUG {
		f, err := os.CreateTemp("/tmp", "log")
		if err == nil {
			l = log.New(f, "", 0)
			l.Println("start")
		}
	}

	if len(config.KeeperRateLimitKey) == 0 {
		locallog("config: config: keeperRateLimitKey is empty")
	}
	if len(config.KeeperURL) == 0 {
		locallog("config: keeperURL is empty")
	}
	if len(config.KeeperAdminPassword) == 0 {
		locallog("config: keeperAdminPassword is empty")
	}
	r := newRateLimit(ctx, next, config, name)
	r.l = l
	return r, nil
}

func (g *GlobalRateLimit) sync() {
	g.umtx.Lock()
	defer g.umtx.Unlock()
	locallog("sync")
	err := grl.setFromSettings()
	if err != nil {
		locallog("cant get ratelimits from keeper: ", err)
	}
}

func (g *GlobalRateLimit) configure(ctx context.Context, config *Config) {
	to := 300 * time.Second
	if du, err := time.ParseDuration(string(config.KeeperReqTimeout)); err == nil {
		to = du
	}
	if ctx != nil {
		i := atomic.AddInt32(g.icnt, 1)
		locallog("run instance. cnt: ", i)
		/*
			go func() {
				<-ctx.Done()
				i := atomic.AddInt32(g.icnt, -1)
				locallog("done instance. cnt: ", i)

				f, err := os.CreateTemp("/tmp", "inst")
				if err == nil {
					f.Close()
				}

				if i == 0 {
				}
			}()
		*/
	}
	g.umtx.Lock()
	defer g.umtx.Unlock()

	if to, err := time.ParseDuration(string(config.KeeperReloadInterval)); err == nil && grl.tickerto != to {
		g.ticker.Reset(to)
		grl.tickerto = to
	}
	g.settings = keeper.New(config.KeeperURL, to, config.KeeperAdminPassword)
	g.config = config
	err := grl.setFromSettings()
	if err != nil {
		if ctx == nil {
			locallog(fmt.Sprintf("init0: keeper: %v. try init from middleware RatelimitData configuration", err))
		} else {
			locallog(fmt.Sprintf("init: keeper: %v. try init from middleware RatelimitData configuration", err))
		}
		err = grl.setFromData()
		//		err = grl.setFromFile()
		if err != nil {
			if ctx == nil {
				locallog(fmt.Sprintf("init0: data: %v", err))
			} else {
				locallog(fmt.Sprintf("init: data: %v", err))
			}
		}
	}
}

func NewRateLimit(next http.Handler, config *Config, name string) *RateLimit {
	return newRateLimit(nil, next, config, name)
}

func newRateLimit(ctx context.Context, next http.Handler, config *Config, name string) *RateLimit {
	r := &RateLimit{
		name: name,
		next: next,
		cnt:  new(int32),
	}
	grl.configure(ctx, config)
	return r
}

func (r *RateLimit) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	encoder := json.NewEncoder(rw)
	if r.allow(req) {
		r.next.ServeHTTP(rw, req)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusTooManyRequests)
	_ = encoder.Encode(map[string]any{"error_code": "ERR_TOO_MANY_REQUESTS", "error_description": "Слишком много запросов. Повторите попытку позднее."})
}

func (r *RateLimit) log(v ...any) {
	if r.l != nil {
		r.l.Println(v...)
	}
}

func locallog(v ...any) {
	_, _ = os.Stderr.WriteString(fmt.Sprintf("time=%q traefikPlugin=\"ratelimit\" msg=%q\n", time.Now().UTC().Format("2006-01-02 15:04:05Z"), fmt.Sprint(v...)))
}
