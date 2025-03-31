package routing

import (
	"gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/campaign"
)

// Config описывает структуру для управления настройками применяемыми для банков.
type Config struct {
	campaign.Config
	// BlackListMembers хранит массив, для которых настройка всегда выключена.
	// Срабатывает только если установлен параметр Enabled
	BlackListMembers []string `json:"black_members"`
	// WhietListBanks хранит массив, для которых настройка всегда включена.
	WhietListBanks []string `json:"white_banks"`
}

type FeatureRoutingDto struct {
	Member string
	UserID string
}

type FeatureBankDto struct {
	Bank   string
	UserID string
}
