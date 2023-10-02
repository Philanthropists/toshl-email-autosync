package bank

import (
	"context"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank/bancolombia"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank/banktypes"
)

type Repository struct{}

func (r Repository) GetBanks(_ context.Context) []banktypes.BankDelegate {
	return []banktypes.BankDelegate{
		bancolombia.Bancolombia{},
	}
}
