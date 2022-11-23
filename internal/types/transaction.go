package types

import (
	"time"
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
	Value   Amount
	Account string
	Date    time.Time
}

type BankDelegate interface {
	FilterMessage(message Message) bool
	ExtractTransactionInfoFromMessage(message Message) (*TransactionInfo, error)
	String() string
}
