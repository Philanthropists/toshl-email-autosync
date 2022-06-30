package types

import (
	"time"

	"github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap/types"
	"github.com/Philanthropists/toshl-go"
)

type Auth struct {
	Addr                 string            `json:"mail-addr"`
	Username             string            `json:"mail-username"`
	Password             string            `json:"mail-password"`
	ToshlToken           string            `json:"toshl-token"`
	TwilioAccountSid     string            `json:"twilio-account-sid"`
	TwilioAuthToken      string            `json:"twilio-auth-token"`
	TwilioFromNumber     string            `json:"twilio-from-number"`
	TwilioToNumber       string            `json:"twilio-to-number"`
	RapidApiKey          string            `json:"rapidapi-key"`
	RapidApiHost         string            `json:"rapidapi-host"`
	ArchiveMailbox       string            `json:"archive_mailbox"`
	Timezone             string            `json:"timezone"`
	StockOptions         StockOptions      `json:"stock_options"`
	FundOptions          FundOptions       `json:"fund_options"`
	ToshlAccountMappings map[string]string `json:"toshl_account_mappings"`
}

type StockOptions struct {
	Enabled bool     `json:"enabled"`
	Stocks  []string `json:"stocks"`
	Times   []string `json:"times"`
}

type FundOptions struct {
	Enabled bool     `json:"enabled"`
	Funds   []string `json:"funds"`
	Times   []string `json:"times"`
}

type Currency struct {
	toshl.Currency
}

type BankMessage struct {
	types.Message

	Bank BankDelegate
}

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
	Bank            BankDelegate
	MsgId           uint32
	TransactionType TransactionType
	Type            string
	Place           string
	Value           Currency
	Account         string
	Date            time.Time
}

type BankDelegate interface {
	FilterMessage(message types.Message) bool
	ExtractTransactionInfoFromMessage(message types.Message) (*TransactionInfo, error)
}
