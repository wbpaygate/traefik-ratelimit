package keeperclient

import (
	"hash/crc64"
	"strconv"

	"gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/campaign"
)

var crc65TableISO = crc64.MakeTable(crc64.ISO)

// userID является вспомогательным типом данных для определения вхождения пользователя в заданные диапазоны.
type userID string

// getAsNumber преобразовывает строковое представление идентификатора пользователя в числовое.
func (uid userID) getAsNumber() uint64 {
	res, err := strconv.ParseUint(string(uid), 10, 64)
	if err == nil {
		return res
	}

	return crc64.Checksum([]byte(uid), crc65TableISO)
}

// checkInList проверяет вхождение идентификатора пользователя в заданный список.
func (uid userID) checkInMap(list map[string]struct{}) bool {
	_, exists := list[string(uid)]
	return exists
}

// IsIncludedInCampaign определяет вхождение идентификатора пользователя в заданную тестовую кампанию.
func (uid userID) IsIncludedInCampaign(cfg *campaign.Config) bool {
	if !cfg.Enabled {
		return false
	}

	if uid.checkInMap(cfg.WhiteListMap) {
		return true
	}

	if uid.checkInMap(cfg.BlackListMap) {
		return false
	}

	if uid.getAsNumber()%100 < cfg.Percent {
		return true
	}

	return false
}
