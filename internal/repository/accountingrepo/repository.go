package accountingrepo

import (
	"context"
	"sync"

	"github.com/Philanthropists/toshl-go"
	"github.com/zeebo/errs"
)

type ToshlClient interface {
	Categories(params *toshl.CategoryQueryParams) ([]toshl.Category, error)
	Accounts(params *toshl.AccountQueryParams) ([]toshl.Account, error)
	CreateCategory(category *toshl.Category) error
	CreateEntry(entry *toshl.Entry) error
}

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

func (r *ToshlRepository) GetAccounts(ctx context.Context, token string) ([]string, error) {
	c := r.getClient(token)

	type response struct {
		Value []toshl.Account
		Err   error
	}

	resp := make(chan response)
	go func() {
		defer close(resp)
		as, err := c.Accounts(nil)
		resp <- response{
			Value: as,
			Err:   err,
		}
	}()

	select {
	case <-ctx.Done():
		return nil, errs.New("context finished: %w", ctx.Err())

	case r := <-resp:
		as, err := r.Value, r.Err
		if err != nil {
			return nil, err
		}

		accounts := make([]string, 0, len(as))
		for _, a := range as {
			accounts = append(accounts, a.Name)
		}

		return accounts, nil
	}
}
