package keeperclient

import "fmt"

func NewPersistentCacheErr(key string) error {
	return fmt.Errorf("persistent cache is empty: %s", key)
}
