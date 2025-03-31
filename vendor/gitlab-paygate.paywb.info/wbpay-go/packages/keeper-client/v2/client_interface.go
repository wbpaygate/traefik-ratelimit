package keeperclient

import (
	"context"

	"gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/routing"
	"gitlab-paygate.paywb.info/wbpay-go/packages/keeper-client/v2/transport"
)

//go:generate go run github.com/golang/mock/mockgen -source=client_interface.go -destination=client_interface_mock.go -package=keeperclient -self_package=.KeeperClient
type KeeperClient interface {
	Get(ctx context.Context, key string) (*transport.Value, error)
	GetFallback(ctx context.Context, key, fallbackValue string) (*transport.Value, bool)
	IsFeatureActiveBank(ctx context.Context, key string, dto routing.FeatureBankDto) (bool, error)
	IsFeatureActiveRouting(ctx context.Context, key string, dto routing.FeatureRoutingDto) (bool, error)
	IsFeatureActive(ctx context.Context, key, user string) (bool, error)
	IsFeatureActiveFallback(ctx context.Context, key, user string, fallbackVal bool) (bool, bool)
	GetAllLocalizationErrors(ctx context.Context) (map[string]map[string]string, error)
	GetAllBankErrors(ctx context.Context, bank string) (map[string]string, error)
}
