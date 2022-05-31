package sync

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Philanthropists/toshl-email-autosync/internal/logger"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/common"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
	"github.com/Philanthropists/toshl-email-autosync/internal/toshl"
	_toshl "github.com/Philanthropists/toshl-go"
)

var localLocation *time.Location

// TODO: obtain this from DynamoDB and set a default value
func SetTimezoneLocale(location string) {
	if location == "" {
		panic("timezone locale should not be empty")
	}

	var err error
	localLocation, err = time.LoadLocation(location)
	if err != nil {
		panic(err)
	}
}

func checkTimezone() {
	if localLocation == nil {
		panic("timezone was not set")
	}
}

func GetMappableAccounts(accounts []*toshl.Account) map[string]*toshl.Account {
	log := logger.GetLogger()
	var exp = regexp.MustCompile(`^(?P<accounts>[0-9\s]+) `)

	var mapping = make(map[string]*toshl.Account)
	for _, account := range accounts {
		name := account.Name
		result := common.ExtractFieldsStringWithRegexp(name, exp)
		if accountNums, ok := result["accounts"]; ok {
			nums := strings.Split(accountNums, " ")
			for _, num := range nums {
				mapping[num] = account
			}

			if len(nums) == 0 {
				log.Warnw("no account found for name",
					"name", name)
			}
		}
	}

	return mapping
}

func CreateInternalCategoryIfAbsent(toshlClient toshl.ApiClient) string {
	const categoryName = "PENDING"

	categories, err := toshlClient.GetCategories()
	if err != nil {
		panic(err)
	}

	for _, c := range categories {
		if c.Name == categoryName {
			return c.ID
		}
	}

	var cat toshl.Category
	cat.Name = categoryName
	cat.Type = "expense"

	err = toshlClient.CreateCategory(&cat)
	if err != nil {
		panic(err)
	}

	return cat.ID
}

func CreateEntries(toshlClient toshl.ApiClient, transactions []*types.TransactionInfo, mappableAccounts map[string]*toshl.Account, internalCategoryId string) ([]*types.TransactionInfo, []*types.TransactionInfo) {
	const DateFormat = "2006-01-02"

	log := logger.GetLogger()

	checkTimezone()

	var successfulTransactions []*types.TransactionInfo
	var failedTransactions []*types.TransactionInfo
	for _, t := range transactions {
		account, ok := mappableAccounts[t.Account]
		if !ok {
			continue
		}

		var newEntry toshl.Entry
		newEntry.Amount = -*t.Value.Rate // negative because it is an expense
		newEntry.Currency = _toshl.Currency{
			Code: "COP",
		}
		newEntry.Date = t.Date.In(localLocation).Format(DateFormat)
		description := fmt.Sprintf("** %s de %s", t.Type, t.Place)
		newEntry.Description = &description
		newEntry.Account = account.ID
		newEntry.Category = internalCategoryId

		err := toshlClient.CreateEntry(&newEntry)
		if err != nil {
			log.Errorf("Failed to create entry for transaction [%+v | %+v]: %s\n", newEntry, t, err)
			failedTransactions = append(failedTransactions, t)
		} else {
			log.Infow("Created entry successfully",
				"entry", newEntry)
			successfulTransactions = append(successfulTransactions, t)
		}
	}

	return successfulTransactions, failedTransactions
}
