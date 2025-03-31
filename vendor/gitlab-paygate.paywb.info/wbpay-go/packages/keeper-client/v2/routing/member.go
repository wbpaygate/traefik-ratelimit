package routing

import (
	"gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/common"
)

// Member является вспомогательным типом данных для определения вхождения банка в заданные диапазоны.
type Member string

// checkInList проверяет вхождение идентификатора банка в заданный список.
func (m Member) checkInList(list []string) bool {
	member := string(m)

	return common.InList(list, member)
}

// IsIncludedInCampaign определяет вхождение идентификатора банка в заданную кампанию.
// Если фича включена, но идентификатора member нет в блэк листе, то фича включена.
func (m Member) IsIncludedInCampaign(cfg *Config) bool {
	if !cfg.Enabled {
		return false
	}

	return !m.checkInList(cfg.BlackListMembers)
}
