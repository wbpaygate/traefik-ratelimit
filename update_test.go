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
		if p, ok := key.(*pattern.Pattern); ok && p.Match([]byte(path)) {
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
		patterns: atomic.Value{},
		headers:  atomic.Value{},
	}

	rl.patterns.Store(&sync.Map{})
	rl.headers.Store(&sync.Map{})

	t.Run("atomic switch with pattern matching", func(t *testing.T) {
		limits := &Limits{
			Limits: []Limit{
				{
					Limit: 10,
					Rules: []Rule{
						{UrlPathPattern: "/api/v1/users"},
						{HeaderKey: "X-User", HeaderVal: "test"},
					},
				},
			},
		}

		rl.hotReloadLimits(limits)

		// проверяем по реальному match'у паттерна
		patternsPtr := rl.patterns.Load()
		if patternsPtr == nil {
			t.Fatalf("patterns is nil")
		}

		patterns, okTypeAssert := patternsPtr.(*sync.Map)
		if !okTypeAssert {
			t.Fatalf("patterns, cannot type assert *sync.Map")
		}

		if _, ok := findPattern(patterns, "/api/v1/users"); !ok {
			t.Error("Initial pattern not found")
		}

		newLimits := &Limits{
			Limits: []Limit{
				{
					Limit: 20,
					Rules: []Rule{
						{UrlPathPattern: "/api/v2/*"},
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
				patterns.Range(func(key, value any) bool { return true })
			}
		}()

		wg.Wait()

		newPatternsPtr := rl.patterns.Load()
		if newPatternsPtr == nil {
			t.Fatalf("newPatterns is nil")
		}

		newPatterns, okTypeAssert := newPatternsPtr.(*sync.Map)
		if !okTypeAssert {
			t.Fatalf("patterns, cannot type assert *sync.Map")
		}

		// проверяем новые лимиты через match паттерна
		if _, ok := findPattern(newPatterns, "/api/v2/users"); !ok {
			t.Error("New pattern not found after reload")
		}

		// проверяем что старые лимиты удалены
		if _, ok := findPattern(newPatterns, "/api/v1/users"); ok {
			t.Error("Old pattern should be removed")
		}
	})

	t.Run("no leaks comprehensive check", func(t *testing.T) {
		oldLimiter1 := limiter.NewLimiter(5)
		oldLimiter2 := limiter.NewLimiter(10)
		oldLimiter3 := limiter.NewLimiter(15)

		patternsPtr := rl.patterns.Load()
		if patternsPtr == nil {
			t.Fatalf("patterns is nil")
		}

		patterns, okTypeAssert := patternsPtr.(*sync.Map)
		if !okTypeAssert {
			t.Fatalf("patterns, cannot type assert *sync.Map")
		}

		patterns.Store(pattern.NewPattern("/path1"), oldLimiter1)
		patterns.Store(pattern.NewPattern("/path2"), oldLimiter3)

		headersPtr := rl.headers.Load()
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
						{UrlPathPattern: "/new-path"},
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

		newPatternsPtr := rl.patterns.Load()
		if newPatternsPtr == nil {
			t.Fatalf("newPatterns is nil")
		}

		newPatterns, okTypeAssert := newPatternsPtr.(*sync.Map)
		if !okTypeAssert {
			t.Fatalf("patterns, cannot type assert *sync.Map")
		}

		// проверяем что новый лимитер установлен и не закрыт
		if newLimiter, ok := findPattern(newPatterns, "/new-path"); ok {
			if newLimiter.IsClosed() {
				t.Error("New limiter should not be closed")
			}

		} else {
			t.Error("New limiter not found")
		}

		// проверяем что старые ключи удалены
		if _, ok := findPattern(newPatterns, "/path1"); ok {
			t.Error("Old pattern /path1 should be removed")
		}
		if _, ok := findPattern(newPatterns, "/path2"); ok {
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
		patternsPtr := rl.patterns.Load()
		if patternsPtr == nil {
			t.Fatalf("patterns is nil")
		}

		patterns, okTypeAssert := patternsPtr.(*sync.Map)
		if !okTypeAssert {
			t.Fatalf("patterns, cannot type assert *sync.Map")
		}

		patterns.Range(func(key, value any) bool {
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
								{UrlPathPattern: fmt.Sprintf("/concurrent/%d", idx)},
							},
						},
					},
				}
				rl.hotReloadLimits(limits)
			}(i)
		}
		wg.Wait()

		count := 0 // после всех обновлений должен остаться последний лимитер
		patternsPtr := rl.patterns.Load()
		if patternsPtr == nil {
			t.Fatalf("patterns is nil")
		}

		patterns, okTypeAssert := patternsPtr.(*sync.Map)
		if !okTypeAssert {
			t.Fatalf("patterns, cannot type assert *sync.Map")
		}

		patterns.Range(func(key, value any) bool {
			count++
			return true
		})

		if count != 1 {
			t.Errorf("Expected 1 pattern, got %d", count)
		}
	})
}
