package keeperclient

import "sync"

// locks выполняет функции блокировки при одновременном обновлении значений в кеше.
type locks struct {
	// keys хранит информацию о локах по каждому отдельному ключу.
	// Если значение установлено в true, в данный момент обновление кеша захвачено одной из горутин.
	keys map[string]bool
	mx   sync.RWMutex
}

// Get возвращает информацию о том идет ли в данный момент обновление конкретного ключа.
func (c *locks) Get(key string) bool {
	c.mx.RLock()
	defer c.mx.RUnlock()

	return c.keys[key]
}

// Set устанавливает блокировку на обновление конкретного ключа другими горутинами.
func (c *locks) Set(key string, value bool) {
	c.mx.Lock()
	c.keys[key] = value
	c.mx.Unlock()
}
