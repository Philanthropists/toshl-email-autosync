package banktypes

import (
	"time"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types/currency"
)

type Message interface {
	ID() uint32
	From() []string
	To() []string
	Subject() string
	Date() time.Time
	Body() []byte
}

type TrxType int8

const (
	Expense TrxType = iota
	Income
	Transaction
)

func (t TrxType) String() string {
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

type TrxInfo struct {
	Date          time.Time
	Bank          BankDelegate
	Action        string
	Description   string
	Account       string
	Value         currency.Amount
	OriginMessage Message
	Type          TrxType
}

type BankDelegate interface {
	ComesFrom(from []string) bool
	FilterMessage(message Message) bool
	ExtractTransactionInfoFromMessage(message Message) (*TrxInfo, error)
	String() string
}
