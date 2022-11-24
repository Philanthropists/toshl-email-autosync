package sync

import (
	"context"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/pkg/pipe"
)

func (s *Sync) ExtractTransactionInfoFromMessages(ctx context.Context, msgs <-chan *types.BankMessage) <-chan pipe.Result[*types.TransactionInfo] {
	txs := pipe.ConcurrentMap(ctx.Done(), s.goroutines(), msgs, func(m *types.BankMessage) (*types.TransactionInfo, error) {
		return m.Bank.ExtractTransactionInfoFromMessage(m.Message)
	})

	return txs
}
