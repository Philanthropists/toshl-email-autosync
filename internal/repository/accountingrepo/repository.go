package accountingrepo

import (
	"context"
	"sync"

	"slices"

	"github.com/Philanthropists/toshl-go"
	"github.com/zeebo/errs"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/repository/accountingrepo/accountingrepotypes"
)

type ToshlClient interface {
	Categories(params *toshl.CategoryQueryParams) ([]toshl.Category, error)
	Accounts(params *toshl.AccountQueryParams) ([]toshl.Account, error)
	CreateCategory(category *toshl.Category) error
	CreateEntry(entry *toshl.Entry) error
}

const (
	Income      = "income"
	Expense     = "expense"
	Transaction = "transaction"
)

type ToshlRepository struct {
	ClientBuilder func(string) ToshlClient

	clients sync.Map
}

func (r *ToshlRepository) getClient(token string) ToshlClient {
	c, ok := r.clients.Load(token)
	if !ok {
		c, _ = r.clients.LoadOrStore(token, r.ClientBuilder(token))
	}

	return c.(ToshlClient)
}

func (r *ToshlRepository) GetAccounts(
	ctx context.Context,
	token string,
) ([]accountingrepotypes.Account, error) {
	c := r.getClient(token)

	return doCancelableOperation(ctx, func() ([]accountingrepotypes.Account, error) {
		ac, err := c.Accounts(nil)
		if err != nil {
			return nil, err
		}

		as := make([]accountingrepotypes.Account, 0, len(ac))
		for _, a := range ac {
			it := accountingrepotypes.Account{
				ID:   a.ID,
				Name: a.Name,
			}
			as = append(as, it)
		}

		return as, nil
	})
}

func (r *ToshlRepository) GetCategories(
	ctx context.Context,
	token string,
) ([]accountingrepotypes.Category, error) {
	c := r.getClient(token)

	return doCancelableOperation(ctx, func() ([]accountingrepotypes.Category, error) {
		cats, err := c.Categories(nil)
		if err != nil {
			return nil, err
		}

		cs := make([]accountingrepotypes.Category, 0, len(cats))
		for _, c := range cats {
			it := accountingrepotypes.Category{
				ID:   c.ID,
				Name: c.Name,
				Type: c.Type,
			}
			cs = append(cs, it)
		}

		return cs, nil
	})
}

func (r *ToshlRepository) CreateCategory(
	ctx context.Context, token, catType, category string,
) (accountingrepotypes.Category, error) {
	c := r.getClient(token)

	validCategoryTypes := []string{
		Income,
		Expense,
		Transaction,
	}

	if !slices.Contains(validCategoryTypes, catType) {
		return accountingrepotypes.Category{}, errs.New("%q is not a valid category", catType)
	}

	cat := toshl.Category{
		Name: category,
		Type: catType,
	}

	err := c.CreateCategory(&cat)
	if err != nil {
		return accountingrepotypes.Category{}, err
	}
	id := accountingrepotypes.Category{
		ID:   cat.ID,
		Name: cat.Name,
		Type: cat.Type,
	}

	return id, nil
}

func (r *ToshlRepository) CreateEntry(
	ctx context.Context, token string, entryInput accountingrepotypes.CreateEntryInput,
) error {
	log := logging.FromContext(ctx)

	c := r.getClient(token)

	const dateFormat = "2006-01-02"

	date := entryInput.Date.Format(dateFormat)
	description := entryInput.Description

	newEntry := toshl.Entry{
		Amount: entryInput.Currency.Number,
		Currency: toshl.Currency{
			Code: entryInput.Currency.Code,
		},
		Date:        date,
		Description: &description,
		Account:     entryInput.AccountID,
		Category:    entryInput.CategoryID,
	}

	log.Debug("entry to create",
		logging.Any("entry", newEntry),
	)

	err := c.CreateEntry(&newEntry)
	if err != nil {
		return errs.New("could not create entry: %w", err)
	}

	return nil
}

func doCancelableOperation[T any](ctx context.Context, op func() (T, error)) (T, error) {
	type response struct {
		Value T
		Err   error
	}

	resp := make(chan response)
	go func() {
		defer close(resp)
		as, err := op()
		resp <- response{
			Value: as,
			Err:   err,
		}
	}()

	var zeroValue T

	select {
	case <-ctx.Done():
		return zeroValue, errs.New("context finished: %w", ctx.Err())

	case r := <-resp:
		val, err := r.Value, r.Err
		if err != nil {
			return zeroValue, err
		}

		return val, nil
	}
}
