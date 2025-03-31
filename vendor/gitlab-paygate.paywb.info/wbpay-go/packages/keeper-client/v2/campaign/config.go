package campaign

// Config описывает структуру для управления настройками применяемыми для части пользователей.
type Config struct {
	// BlackList хранит массив пользователей, для которых настройка всегда выключена.
	// Срабатывает только если установлен параметр Enabled
	BlackList []string `json:"black_list"`
	// WhiteList - хранит массив пользователей, для которых настройка всегда включена.
	// Срабатывает только если установлен параметр Enabled
	WhiteList []string `json:"white_list"`
	// Percent хранит процент пользователей, для которых настройка всегда выключена.
	// Срабатывает только если установлен параметр Enabled
	Percent uint64 `json:"percent"`
	// Enabled - глобальное состояние параметра.
	Enabled bool `json:"enabled"`

	BlackListMap map[string]struct{} `json:"-"`
	WhiteListMap map[string]struct{} `json:"-"`
}
