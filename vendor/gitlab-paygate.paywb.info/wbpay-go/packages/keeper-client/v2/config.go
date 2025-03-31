package keeperclient

import (
	"time"
)

// Config описывает структуру конфигурации для подключения к сервису keeper.
type Config struct {
	// KeeperURL базовый адрес сервиса (без указания путей)
	KeeperURL   string `envconfig:"KEEPER_URL" default:"http://keeper-ext:8080"`
	KeeperProxy string `envconfig:"KEEPER_PROXY" default:""`
	// KeeperSettingsPath Для запросов в megakeeper задать /api/v1/settings
	KeeperSettingsPath string `envconfig:"KEEPER_SETTINGS_PATH" default:"admin/get"`
	// KeeperSettingsAllPath Для запросов в megakeeper задать /api/v1/settings
	KeeperSettingsAllPath string `envconfig:"KEEPER_SETTINGS_ALL_PATH" default:"admin/get_all"`
	// ReqTimeout устанавливает максимальное время запроса при получении параметров
	ReqTimeout time.Duration `envconfig:"KEEPER_REQ_TIMEOUT" default:"300s"`
	// CacheTTL позволяет установить время для кеширования значений запрашиваемых параметров
	CacheTTL time.Duration `envconfig:"KEEPER_CACHE_TTL"`
}
