package sync

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	toshl "github.com/Philanthropists/toshl-email-autosync/v2/internal/store/saas/toshl/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types"
	util "github.com/Philanthropists/toshl-email-autosync/v2/internal/util/regexp"
	"github.com/Philanthropists/toshl-email-autosync/v2/pkg/pipe"
	"go.uber.org/zap"
)

type toshlClient interface {
	GetAccounts() ([]*toshl.Account, error)
	CreateEntry(entry *toshl.Entry) error
	GetCategories() ([]*toshl.Category, error)
	CreateCategory(category *toshl.Category) error
}

func (s *Sync) SaveTransactions(ctx context.Context, client toshlClient, txs <-chan *types.TransactionInfo) <-chan pipe.Result[*types.TransactionInfo] {

	cache := toshlCache{
		Client: client,
	}

	f := func(t *types.TransactionInfo) (*types.TransactionInfo, error) {
		_, err := s.createEntry(client, &cache, t)
		return t, err
	}

	return pipe.ConcurrentMap(ctx.Done(), s.goroutines(), txs, f)
}

type toshlCache struct {
	Client           toshlClient
	cats             map[types.TransactionType]string
	mappableAccounts struct {
		once sync.Once
		m    map[string]string
	}
}

func (cs *toshlCache) createCategoryIfAbsent(t types.TransactionType) (string, error) {
	const categoryNamePrefix = "PENDING_"

	// sanity check
	if !t.IsValid() {
		return "", errors.New("transaction type is invalid")
	}

	categoryName := categoryNamePrefix + strings.ToUpper(t.String())

	categories, err := cs.Client.GetCategories()
	if err != nil {
		return "", fmt.Errorf("could not get categories: %w", err)
	}

	for _, c := range categories {
		if c.Name == categoryName {
			return c.ID, nil
		}
	}

	var cat toshl.Category
	cat.Name = categoryName
	cat.Type = t.String()

	err = cs.Client.CreateCategory(&cat)
	if err != nil {
		return "", fmt.Errorf("could not create category for %s: %w", t.String(), err)
	}

	return cat.ID, nil

}

func (cs *toshlCache) GetCategory(t types.TransactionType) (string, error) {
	if cs.cats == nil {
		cs.cats = make(map[types.TransactionType]string)
	}

	if cs.Client == nil {
		return "", errors.New("client is nil")
	}

	_, ok := cs.cats[t]
	if !ok {
		id, err := cs.createCategoryIfAbsent(t)
		if err != nil {
			return "", err
		}

		cs.cats[t] = id
	}

	return cs.cats[t], nil
}

func (cs *toshlCache) buildAccountsCache() error {
	var exp = regexp.MustCompile(`^(?P<accounts>[0-9\s]+) `)

	accounts, err := cs.Client.GetAccounts()
	if err != nil {
		return err
	}

	var mapping = make(map[string]string)
	for _, account := range accounts {
		name := account.Name
		result := util.ExtractFields(name, exp)

		if accountNums, ok := result["accounts"]; ok {
			nums := strings.Split(accountNums, " ")
			for _, n := range nums {
				mapping[n] = account.ID
			}
		}
	}

	cs.mappableAccounts.m = mapping

	return nil
}

func (cs *toshlCache) GetMappableAccount(id string) (string, error) {
	cs.mappableAccounts.once.Do(func() {
		err := cs.buildAccountsCache()
		if err != nil {
			log.Panicf("could not generate accounts mapping cache: %v", err)
		}
	})

	id, ok := cs.mappableAccounts.m[id]
	if !ok {
		return "", fmt.Errorf("mappable account not found for [%s]", id)
	}

	return id, nil
}

func (s *Sync) createEntry(client toshlClient, cache *toshlCache, tx *types.TransactionInfo) (*toshl.Entry, error) {
	const DateFormat = "2006-01-02"

	accountID, err := cache.GetMappableAccount(tx.Account)
	if err != nil {
		return nil, err
	}

	categoryID, err := cache.GetCategory(tx.Type)
	if err != nil {
		return nil, err
	}

	var newEntry toshl.Entry
	newEntry.Amount = tx.Value.Number
	if tx.Type == types.Expense {
		newEntry.Amount *= -1 // negative if it is an expense
	}

	var cur toshl.Currency
	cur.Code = "COP"

	newEntry.Currency = cur.Currency

	location, err := time.LoadLocation(s.Config.Timezone)
	if err != nil {
		panic(err)
	}

	newEntry.Date = tx.Date.In(location).Format(DateFormat)
	description := fmt.Sprintf("** %s de %s", tx.Type, tx.Place)
	newEntry.Description = &description
	newEntry.Account = accountID
	newEntry.Category = categoryID

	s.log().Debug("entry to create", zap.Reflect("entry", newEntry))

	err = client.CreateEntry(&newEntry)
	if err != nil {
		return nil, fmt.Errorf("could not create entry: %w", err)
	}

	return &newEntry, nil
}
