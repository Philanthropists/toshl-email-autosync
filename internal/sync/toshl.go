package sync

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Philanthropists/toshl-email-autosync/internal/logger"
	"github.com/Philanthropists/toshl-email-autosync/internal/notifications"
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

func CreateInternalCategoryIfAbsent(toshlClient toshl.ApiClient, TxType types.TransactionType) string {
	const categoryNamePrefix = "PENDING_"

	// sanity check
	if !TxType.IsValid() {
		panic("transaction type is invalid")
	}

	categoryName := categoryNamePrefix + strings.ToUpper(TxType.String())

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
	cat.Type = TxType.String()

	err = toshlClient.CreateCategory(&cat)
	if err != nil {
		panic(err)
	}

	return cat.ID
}

func CreateEntries(toshlClient toshl.ApiClient, transactions []*types.TransactionInfo, mappableAccounts map[string]*toshl.Account, internalCategoryIds map[types.TransactionType]string) ([]*types.TransactionInfo, []*types.TransactionInfo) {
	log := logger.GetLogger()

	checkTimezone()

	var successfulTransactions []*types.TransactionInfo
	var failedTransactions []*types.TransactionInfo
	for _, tx := range transactions {
		newEntry, err := createEntry(toshlClient, tx, mappableAccounts, internalCategoryIds)

		if err != nil {
			log.Errorf("Failed to create entry for transaction [%+v | %+v]: %s\n", newEntry, tx, err)
			_ = notifications.PushNotification(fmt.Sprintf("ERROR: %s", err))
			failedTransactions = append(failedTransactions, tx)
		} else {
			log.Infow("Created entry successfully",
				"entry", newEntry)
			successfulTransactions = append(successfulTransactions, tx)
		}
	}

	return successfulTransactions, failedTransactions
}

func createEntry(toshlClient toshl.ApiClient, tx *types.TransactionInfo, mappableAccounts map[string]*toshl.Account, internalCategoryIds map[types.TransactionType]string) (*toshl.Entry, error) {
	const DateFormat = "2006-01-02"

	account, err := getAccountFromMappableAccounts(mappableAccounts, tx.Account)
	if err != nil {
		return nil, err
	}

	txType := tx.TransactionType
	internalCategoryId, err := getInternalCategoryIdFromTxType(internalCategoryIds, txType)
	if err != nil {
		return nil, err
	}

	var newEntry toshl.Entry
	newEntry.Amount = *tx.Value.Rate
	if txType == types.Expense {
		newEntry.Amount *= -1 // negative if it is an expense
	}

	newEntry.Currency = _toshl.Currency{
		Code: "COP",
	}

	newEntry.Date = tx.Date.In(localLocation).Format(DateFormat)
	description := fmt.Sprintf("** %s de %s", tx.Type, tx.Place)
	newEntry.Description = &description
	newEntry.Account = account.ID
	newEntry.Category = internalCategoryId

	err = toshlClient.CreateEntry(&newEntry)
	if err != nil {
		return nil, err
	}

	return &newEntry, nil
}

func getAccountFromMappableAccounts(mappableAccounts map[string]*toshl.Account, accountName string) (*toshl.Account, error) {
	account, ok := mappableAccounts[accountName]
	if !ok {
		return nil, fmt.Errorf(`could not find account [%s] in available accounts`, accountName)
	}

	return account, nil
}

func getInternalCategoryIdFromTxType(internalCategoryIds map[types.TransactionType]string, txType types.TransactionType) (string, error) {
	internalCategoryId, ok := internalCategoryIds[txType]
	if !ok {
		return "", fmt.Errorf(`could not get internal category id for [%s]`, txType)
	}

	return internalCategoryId, nil
}
