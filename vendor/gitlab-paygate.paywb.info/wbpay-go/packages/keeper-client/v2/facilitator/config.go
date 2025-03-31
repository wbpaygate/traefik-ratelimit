package facilitator

import "gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/campaign"

type FcSchemeBlock struct {
	// PaymentSystem набор платежных систем, может быть несколько
	PaymentSystem map[string]bool `json:"ps"`
}

// FsScheme Параметр для включения фасилитаторской схемы в банках по параметрам.
type FsScheme struct {
	campaign.Config
	// Self название своего эмитента
	Self string `json:"self"`
	// Onus блок условий "сам на себя"
	Onus FcSchemeBlock `json:"onus"`
	// Offus блок условий "чужая эмиссия"
	Offus FcSchemeBlock `json:"offus"`
}
