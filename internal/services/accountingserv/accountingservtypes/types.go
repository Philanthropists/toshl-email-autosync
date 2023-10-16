package accountingservtypes

import (
	"time"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types/currency"
)

type Account struct {
	ID   string
	Name string
}

type Category struct {
	ID   string
	Name string
	Type string
}

type CreateEntryInput struct {
	Date        time.Time
	Currency    currency.Amount
	Description string
	AccountID   string
	CategoryID  string
}
