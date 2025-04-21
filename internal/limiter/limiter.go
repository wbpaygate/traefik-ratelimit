package limiter

import (
	"sync/atomic"
	"time"
)

const WindowCount = 5

type Limiter struct {
	limit    atomic.Int32
	windows  []atomic.Int32 // кольцевой буфер окон (сегментов секунды)
	shutdown atomic.Int32   // флаг для остановки фонового обновления окон
}

func NewLimiter(limit int) *Limiter {
	limit32 := int32(limit)

	l := &Limiter{
		limit:    atomic.Int32{},
		shutdown: atomic.Int32{},
		windows:  make([]atomic.Int32, WindowCount),
	}

	// инициализируем все окна начальным значением лимита
	for i := 0; i < WindowCount; i++ {
		l.windows[i] = atomic.Int32{}
		l.windows[i].Store(limit32 / WindowCount)
	}

	l.limit.Store(limit32)

	go l.backgroundWindowReset()
	return l
}

// backgroundWindowReset фон горутина для сброса окон по таймеру
func (l *Limiter) backgroundWindowReset() {
	for l.shutdown.Load() == 0 {
		// вычисляем индекс окна, которое нужно сбросить
		// добавляем WindowCount/2 для равномерного распределения сбросов
		windowToReset := (time.Now().Second() + (WindowCount / 2)) % WindowCount
		l.windows[windowToReset].Store(l.limit.Load())
		time.Sleep(time.Second)
	}
}

func (l *Limiter) Close() {
	l.shutdown.Store(1)
}

func (l *Limiter) IsClosed() bool {
	if val := l.shutdown.Load(); val > 0 {
		return true
	}

	return false
}

func (l *Limiter) Limit() int {
	return int(l.limit.Load())
}

func (l *Limiter) Allow() bool {
	if l.limit.Load() <= 0 {
		return true
	}

	currentWindow := time.Now().Second() % WindowCount
	// уменьшаем счётчик окна и проверяем результат
	a := l.windows[currentWindow].Add(-1) >= 0
	return a
}
