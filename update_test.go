package traefik_ratelimit

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wbpaygate/traefik-ratelimit/internal/limiter"
	"github.com/wbpaygate/traefik-ratelimit/internal/pattern"
)

// пспомогательная функция для поиска паттерна в sync.Map
var findPattern = func(m *sync.Map, path string) (*limiter.Limiter, bool) {
	var foundLimiter *limiter.Limiter
	var found bool

	m.Range(func(key, value any) bool {
		if rule, ok := key.(RuleImpl); ok && rule.URLPathPattern.Match([]byte(path)) {
			foundLimiter = value.(*limiter.Limiter)
			found = true

			return false
		}

		return true
	})

	return foundLimiter, found
}

func TestRateLimiter_hotReloadLimits(t *testing.T) {
	rl := &RateLimiter{
		rules: atomic.Value{},
	}

	rl.rules.Store(&sync.Map{})

	t.Run("atomic switch with pattern matching", func(t *testing.T) {
		limits := &Limits{
			Limits: []Limit{
				{
					Limit: 10,
					Rules: []Rule{
						{URLPathPattern: "/api/v1/users", HeaderKey: "X-User", HeaderVal: "test"},
					},
				},
			},
		}

		rl.hotReloadLimits(limits)

		rulesPtr := rl.rules.Load()
		if rulesPtr == nil {
			t.Fatalf("rules is nil")
		}

		rules, okTypeAssert := rulesPtr.(*sync.Map)
		if !okTypeAssert {
			t.Fatalf("rules, cannot type assert *sync.Map")
		}

		if _, ok := findPattern(rules, "/api/v1/users"); !ok {
			t.Error("Initial pattern not found")
		}

		newLimits := &Limits{
			Limits: []Limit{
				{
					Limit: 20,
					Rules: []Rule{
						{URLPathPattern: "/api/v2/*"},
					},
				},
			},
		}

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			rl.hotReloadLimits(newLimits)
		}()

		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				rules.Range(func(key, value any) bool { return true })
			}
		}()

		wg.Wait()

		newRulesPtr := rl.rules.Load()
		if newRulesPtr == nil {
			t.Fatalf("newRulesPtr is nil")
		}

		newRules, okTypeAssert := newRulesPtr.(*sync.Map)
		if !okTypeAssert {
			t.Fatalf("rules, cannot type assert *sync.Map")
		}

		// проверяем новые лимиты через match паттерна
		if _, ok := findPattern(newRules, "/api/v2/users"); !ok {
			t.Error("New rules not found after reload")
		}

		// проверяем что старые лимиты удалены
		if _, ok := findPattern(newRules, "/api/v1/users"); ok {
			t.Error("Old rules should be removed")
		}
	})

	t.Run("no leaks comprehensive check", func(t *testing.T) {
		oldLimiter1 := limiter.NewLimiter(5)
		oldLimiter2 := limiter.NewLimiter(10)
		oldLimiter3 := limiter.NewLimiter(15)

		rulesPtr := rl.rules.Load()
		if rulesPtr == nil {
			t.Fatalf("rulesPtr is nil")
		}

		rules, okTypeAssert := rulesPtr.(*sync.Map)
		if !okTypeAssert {
			t.Fatalf("rules, cannot type assert *sync.Map")
		}

		rules.Store(pattern.NewPattern("/path1"), oldLimiter1)
		rules.Store(pattern.NewPattern("/path2"), oldLimiter3)

		headersPtr := rl.rules.Load()
		if headersPtr == nil {
			t.Fatalf("headers is nil")
		}

		headers, okTypeAssert := headersPtr.(*sync.Map)
		if !okTypeAssert {
			t.Fatalf("headers, cannot type assert *sync.Map")
		}

		headers.Store(&Header{key: "X-Test", val: "1"}, oldLimiter2)

		newLimits := &Limits{
			Limits: []Limit{
				{
					Limit: 20,
					Rules: []Rule{
						{URLPathPattern: "/new-path"},
					},
				},
			},
		}

		rl.hotReloadLimits(newLimits)
		time.Sleep(time.Second)

		if !oldLimiter1.IsClosed() {
			t.Error("Old limiter 1 was not closed")
		}
		if !oldLimiter2.IsClosed() {
			t.Error("Old limiter 2 was not closed")
		}
		if !oldLimiter3.IsClosed() {
			t.Error("Old limiter 3 was not closed")
		}

		newRulesPtr := rl.rules.Load()
		if newRulesPtr == nil {
			t.Fatalf("newRules is nil")
		}

		newRules, okTypeAssert := newRulesPtr.(*sync.Map)
		if !okTypeAssert {
			t.Fatalf("rules, cannot type assert *sync.Map")
		}

		// проверяем что новый лимитер установлен и не закрыт
		if newLimiter, ok := findPattern(newRules, "/new-path"); ok {
			if newLimiter.IsClosed() {
				t.Error("New limiter should not be closed")
			}

		} else {
			t.Error("New limiter not found")
		}

		// проверяем что старые ключи удалены
		if _, ok := findPattern(newRules, "/path1"); ok {
			t.Error("Old pattern /path1 should be removed")
		}
		if _, ok := findPattern(newRules, "/path2"); ok {
			t.Error("Old pattern /path2 should be removed")
		}
	})

	t.Run("empty rules handling", func(t *testing.T) {
		emptyLimits := &Limits{
			Limits: []Limit{
				{
					Limit: 5,
					Rules: []Rule{}, // пустые правила
				},
			},
		}

		rl.hotReloadLimits(emptyLimits) // не должно паниковать

		count := 0 // проверяем что мапы пусты
		rulesPtr := rl.rules.Load()
		if rulesPtr == nil {
			t.Fatalf("rules is nil")
		}

		rules, okTypeAssert := rulesPtr.(*sync.Map)
		if !okTypeAssert {
			t.Fatalf("rules, cannot type assert *sync.Map")
		}

		rules.Range(func(key, value any) bool {
			count++
			return true
		})

		if count > 0 {
			t.Error("Patterns map should be empty")
		}
	})

	t.Run("concurrent reloads", func(t *testing.T) {
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				limits := &Limits{
					Limits: []Limit{
						{
							Limit: idx + 1,
							Rules: []Rule{
								{URLPathPattern: fmt.Sprintf("/concurrent/%d", idx)},
							},
						},
					},
				}
				rl.hotReloadLimits(limits)
			}(i)
		}
		wg.Wait()

		count := 0 // после всех обновлений должен остаться последний лимитер
		rulesPtr := rl.rules.Load()
		if rulesPtr == nil {
			t.Fatalf("rules is nil")
		}

		rules, okTypeAssert := rulesPtr.(*sync.Map)
		if !okTypeAssert {
			t.Fatalf("rules, cannot type assert *sync.Map")
		}

		rules.Range(func(key, value any) bool {
			count++
			return true
		})

		if count != 1 {
			t.Errorf("Expected 1 pattern, got %d", count)
		}
	})
}
