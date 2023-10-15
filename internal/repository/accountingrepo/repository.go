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

	return doCancelableOperation[[]string](ctx, func() ([]string, error) {
		ac, err := c.Accounts(nil)
		if err != nil {
			return nil, err
		}

		as := make([]string, 0, len(ac))
		for _, a := range ac {
			as = append(as, a.Name)
		}

		return as, nil
	})
}

func (r *ToshlRepository) GetCategories(ctx context.Context, token string) ([]string, error) {
	c := r.getClient(token)

	return doCancelableOperation[[]string](ctx, func() ([]string, error) {
		cats, err := c.Categories(nil)
		if err != nil {
			return nil, err
		}

		cs := make([]string, 0, len(cats))
		for _, c := range cats {
			cs = append(cs, c.Name)
		}

		return cs, nil
	})
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
