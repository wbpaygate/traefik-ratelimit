package transport

import (
	"context"
	"fmt"
)

// Transport описывает ожидаемое поведение транспортного уровня.
type Transport interface {
	// Get возвращает результат запроса к сервису.
	Get(ctx context.Context, key string) (*Value, error)

	// GetAllSettings возвращает все параметры.
	GetAllSettings(ctx context.Context) ([]ExtendedValue, error)

	// GetAllLocalizationErrors - возвращает мапу ошибок.
	GetAllLocalizationErrors(ctx context.Context) (map[string]map[string]string, error)

	GetAllBankErrors(ctx context.Context, bank string) (map[string]string, error)
}

// Value описывает структуру данных, которая ожидается как результат работы транспортного уровня.
type Value struct {
	// Value хранит исходное значение параметра в сервисе keeper.
	Value string `json:"value"`
	// Version хранит текущую версию ключа. Удаление приводит к сбросу версии в 0.
	// Модификация значения приводит к увеличению данного значения.
	Version int64 `json:"version,omitempty"`
	// Время обновления записи
	ModRevision int64 `json:"mod_revision,omitempty"`
}

type ExtendedValue struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Version     int64  `json:"version"`
	Author      string `json:"author"`
	Description string `json:"description"`
	ModRevision int64  `json:"mod_revision"`
}

type ValuesStore map[string]ExtendedValue

func NewBadStatusCodeErr(code int, url string) error {
	return fmt.Errorf("keeper service returned unexpected status code: %d; url: %s", code, url)
}
