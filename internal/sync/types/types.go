package types

import (
	"github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap/types"
	"github.com/Philanthropists/toshl-go"
	"time"
)

type Auth struct {
	Addr             string `json:"mail-addr"`
	Username         string `json:"mail-username"`
	Password         string `json:"mail-password"`
	ToshlToken       string `json:"toshl-token"`
	TwilioAccountSid string `json:"twilio-account-sid"`
	TwilioAuthToken  string `json:"twilio-auth-token"`
	TwilioFromNumber string `json:"twilio-from-number"`
	TwilioToNumber   string `json:"twilio-to-number"`
}

type Currency struct {
	toshl.Currency
}

type BankMessage struct {
	types.Message

	Bank BankDelegate
}

type TransactionInfo struct {
	Bank    BankDelegate
	MsgId   uint32
	Type    string
	Place   string
	Value   Currency
	Account string
	Date    time.Time
}

type BankDelegate interface {
	FilterMessage(message types.Message) bool
	ExtractTransactionInfoFromMessage(message types.Message) (*TransactionInfo, error)
}
