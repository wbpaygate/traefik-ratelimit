package routing

import (
	"gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/common"
)

// Bank является вспомогательным типом данных для определения вхождения банка в заданные диапазоны.
type Bank string

// checkInList проверяет вхождение банка в заданный список.
func (m Bank) checkInList(list []string) bool {
	bank := string(m)

	return common.InList(list, bank)
}

// IsIncludedInCampaign определяет вхождение банка в заданную кампанию.
// Если банка есть в листе, то фича включена.
func (m Bank) IsIncludedInCampaign(cfg *Config) bool {
	return m.checkInList(cfg.WhietListBanks)
}
