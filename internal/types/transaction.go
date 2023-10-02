package types

import (
	"time"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types/currency"
)

type TransactionType int8

const (
	Expense TransactionType = iota
	Income
	Transaction
)

func (t TransactionType) String() string {
	switch t {
	case Expense:
		return "expense"
	case Income:
		return "income"
	case Transaction:
		return "transaction"
	default:
		return "undefined"
	}
}

func (t TransactionType) IsValid() bool {
	return t.String() != "undefined"
}

type TransactionInfo struct {
	Bank    BankDelegate
	Type    TransactionType
	MsgId   uint32
	Action  string
	Place   string
	Value   currency.Amount
	Account string
	Date    time.Time
}

type BankDelegate interface {
	ComesFrom(from []string) bool
	FilterMessage(message Message) bool
	ExtractTransactionInfoFromMessage(message Message) (*TransactionInfo, error)
	String() string
}
