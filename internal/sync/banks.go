package sync

import (
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank/bancolombia"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types"
)

func (s Sync) AvailableBanks() []types.BankDelegate {
	return []types.BankDelegate{
		bancolombia.Bancolombia{},
	}
}
