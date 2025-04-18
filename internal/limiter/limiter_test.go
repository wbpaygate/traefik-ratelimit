package limiter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLimiter_Allow_Negative(t *testing.T) {
	t.Run("zero limit - all requests allow", func(t *testing.T) {
		l := NewLimiter(0)
		defer l.Close()

		if !l.Allow() {
			t.Error("Allow() should return true for zero limit")
		}
	})

	t.Run("negative limit - all requests allow", func(t *testing.T) {
		l := NewLimiter(-1)
		defer l.Close()

		if !l.Allow() {
			t.Error("Allow() should return true for negative limit")
		}
	})

	t.Run("burst behavior", func(t *testing.T) {
		const limit = 10
		l := NewLimiter(limit)
		defer l.Close()

		allowed := 0
		for i := 0; i < limit*2; i++ { // пробуем сделать в 2 раза больше запросов
			if l.Allow() {
				allowed++
			}
		}

		t.Logf("Initial burst: allowed %d/%d", allowed, limit)
		if allowed != limit {
			t.Errorf("Initial burst: got %d allowed, want %d", allowed, limit)
		}

		// проверяем что новые запросы отклоняются
		if l.Allow() {
			t.Error("Request after burst was allowed, want denied")
		}

		time.Sleep(1100 * time.Millisecond)

		// проверяем новый бурс
		allowed = 0
		for i := 0; i < limit*2; i++ {
			if l.Allow() {
				allowed++
			}
		}

		t.Logf("New burst after refresh: allowed %d/%d", allowed, limit)
		if allowed != limit {
			t.Errorf("New burst: got %d allowed, want %d", allowed, limit)
		}
	})
}

func TestLimiter_Allow_Concurrent(t *testing.T) {
	const limit = 100
	const workers = 10
	const requestsPerWorker = 50

	l := NewLimiter(limit)
	defer l.Close()

	var allowed int32
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerWorker; j++ {
				if l.Allow() {
					atomic.AddInt32(&allowed, 1)
				}
			}
		}()
	}

	wg.Wait()

	if allowed > limit {
		t.Errorf("Allowed %d requests, want no more than %d", allowed, limit)
	}
}

//go test -bench=. -benchmem ./internal/limiter
//goos: darwin
//goarch: arm64
//pkg: github.com/wbpaygate/traefik-ratelimit/internal/limiter
//cpu: Apple M3 Pro
//BenchmarkLimiter/LowLimit-SingleThread-12               26666360                44.88 ns/op            167.1 ops/s             0 B/op          0 allocs/op
//BenchmarkLimiter/HighLimit-SingleThread-12              26611348                44.97 ns/op          16712 ops/s               0 B/op          0 allocs/op
//BenchmarkLimiter/LowLimit-10Threads-12                  17526339                68.35 ns/op            167.0 ops/s             0 B/op          0 allocs/op
//BenchmarkLimiter/HighLimit-10Threads-12                 17332774                66.51 ns/op          26024 ops/s               0 B/op          0 allocs/op
//BenchmarkLimiter/SmallWindow-100Threads-12              15221858                83.56 ns/op           1572 ops/s               0 B/op          0 allocs/op
//BenchmarkLimiter/LargeWindow-100Threads-12              17642766                82.73 ns/op           6852 ops/s               0 B/op          0 allocs/op
//BenchmarkAllowPure-12                                   15085152                75.91 ns/op            0 B/op          0 allocs/op
//BenchmarkRealistic-12                                   13807704                80.51 ns/op            0 B/op          0 allocs/op

func BenchmarkLimiter(b *testing.B) {
	scenarios := []struct {
		name       string
		windowSize int
		limit      int
		goroutines int
	}{
		{"LowLimit-SingleThread", 10, 100, 1},
		{"HighLimit-SingleThread", 10, 10000, 1},
		{"LowLimit-10Threads", 10, 100, 10},
		{"HighLimit-10Threads", 10, 10000, 10},
		{"SmallWindow-100Threads", 5, 1000, 100},
		{"LargeWindow-100Threads", 60, 5000, 100},
	}

	for _, sc := range scenarios {
		b.Run(sc.name, func(b *testing.B) {
			limiter := NewLimiter(sc.limit)
			defer limiter.Close()

			var ops atomic.Int64

			var wg sync.WaitGroup
			wg.Add(sc.goroutines)

			for i := 0; i < sc.goroutines; i++ {
				go func() {
					defer wg.Done()
					for n := 0; n < b.N/sc.goroutines; n++ {
						if limiter.Allow() {
							ops.Add(1)
						}
					}
				}()
			}

			wg.Wait()

			b.ReportMetric(float64(ops.Load())/b.Elapsed().Seconds(), "ops/s")
		})
	}
}

func BenchmarkAllowPure(b *testing.B) {
	limiter := NewLimiter(100000)
	defer limiter.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow() // тестируем только скорость
		}
	})
}

func BenchmarkRealistic(b *testing.B) {
	limiter := NewLimiter(1000)
	defer limiter.Close()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if limiter.Allow() {
				time.Sleep(100 * time.Microsecond) // симулируем полезную нагрузку
			}
		}
	})
}
