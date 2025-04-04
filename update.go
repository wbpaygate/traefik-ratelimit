package traefik_ratelimit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
)

func (g *GlobalRateLimit) logWorkingLimits() {
	buf := new(bytes.Buffer)
	if err := json.Compact(buf, g.rawlimits); err != nil {
		g.wrapLogger.Info(fmt.Sprintf("working limits: %s", g.rawlimits))

	} else {
		g.wrapLogger.Info(fmt.Sprintf("working limits: %s", buf.String()))
	}

	limits := g.limits[int(atomic.LoadInt32(grl.curlimit))].limits

	for p, lim := range limits {
		if lim.limit != nil {
			g.wrapLogger.Info(fmt.Sprintf("working limit rule %d,%d: %q \"\" \"\" %p %p %d %d", g.version.Version, g.version.ModRevision, p, lim.limit, lim.limit.limiter, lim.limit.Limit, lim.limit.limiter.Limit()))
		}
		for _, lim2 := range lim.limits {
			for val, lim3 := range lim2.limits {
				g.wrapLogger.Info(fmt.Sprintf("working limit rule %d,%d: %q %q %q %p %p %d %d", g.version.Version, g.version.ModRevision, p, lim2.key, val, lim3, lim3.limiter, lim3.Limit, lim3.limiter.Limit()))
			}
		}
	}
}

func (g *GlobalRateLimit) setFromFile() error {
	defer g.logWorkingLimits()
	if g.config == nil {
		return fmt.Errorf("config not specified")
	}

	b, err := os.ReadFile(g.config.RatelimitPath)
	if err != nil {
		return err
	}

	err = g.update(b)
	if err == nil {
		g.rawlimits = b
		g.version.Version = 0
		g.version.ModRevision = 0
	}

	return err
}

func (g *GlobalRateLimit) setFromData() error {
	defer g.logWorkingLimits()
	if g.config == nil {
		return fmt.Errorf("config not specified")
	}

	b := []byte(g.config.RatelimitData)

	err := g.update(b)
	if err == nil {
		g.rawlimits = b
		g.version.Version = 0
		g.version.ModRevision = 0
	}

	return err
}

func (g *GlobalRateLimit) setFromSettings(ctx context.Context) error {
	if g.config == nil {
		g.logWorkingLimits()
		return fmt.Errorf("config not specified")
	}

	result, err := g.GetSettings(ctx)
	if err != nil {
		g.logWorkingLimits()
		return err
	}

	if result == nil || len(result.Value) == 0 {
		g.logWorkingLimits()
		return fmt.Errorf("settings not found in keeper")
	}

	if !g.version.Equal(result) {
		defer g.logWorkingLimits()

		if g.version != nil {
			g.wrapLogger.Info(fmt.Sprintf("old configuration: Version: %d, ModRevision: %d", g.version.Version, g.version.ModRevision))
		}

		err = g.update([]byte(result.Value))
		if err != nil {
			return err
		}

		g.rawlimits = []byte(result.Value)
		g.version = result
		g.wrapLogger.Info(fmt.Sprintf("new configuration loaded: Version: %d, ModRevision: %d", g.version.Version, g.version.ModRevision))
	}

	return nil
}

func (r *RateLimit) Update(b []byte) error {
	return grl.update(b)
}
