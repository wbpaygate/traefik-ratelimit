package keeperclient

// Value copy from gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/transport
type Value struct {
	// Value хранит исходное значение параметра в сервисе keeper.
	Value string `json:"value"`
	// Version хранит текущую версию ключа. Удаление приводит к сбросу версии в 0.
	// Модификация значения приводит к увеличению данного значения.
	Version int64 `json:"version,omitempty"`
	// Время обновления записи
	ModRevision int64 `json:"mod_revision,omitempty"`
}

func (v *Value) Equal(v2 *Value) bool {
	if v == nil || v2 == nil {
		return false
	}

	return v.Version == v2.Version && v.ModRevision == v2.ModRevision
}
