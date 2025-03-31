package keeperclient

import (
	"context"

	"github.com/pkg/errors"
)

func (c *Client) GetAllLocalizationErrors(ctx context.Context) (map[string]map[string]string, error) {
	value, err := c.transport.GetAllLocalizationErrors(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "could not get value from transport")
	}

	return value, nil
}

func (c *Client) GetAllBankErrors(ctx context.Context, bank string) (map[string]string, error) {
	value, err := c.transport.GetAllBankErrors(ctx, bank)
	if err != nil {
		return nil, errors.Wrap(err, "could not get value from transport")
	}

	return value, nil
}
