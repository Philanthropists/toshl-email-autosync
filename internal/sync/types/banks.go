package types

import "github.com/Philanthropists/toshl-email-autosync/v2/internal/types"

type BankDelegate interface {
	FilterMessage(message types.Message) bool
	ExtractTransactionInfoFromMessage(message types.Message) (*types.TransactionInfo, error)
	Name() string
}
