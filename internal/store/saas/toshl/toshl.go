package toshl

import (
	"errors"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/store/saas/toshl/types"
	_toshl "github.com/Philanthropists/toshl-go"
)

type toshlClient interface {
	Categories(params *_toshl.CategoryQueryParams) ([]_toshl.Category, error)
	Accounts(params *_toshl.AccountQueryParams) ([]_toshl.Account, error)
	CreateCategory(category *_toshl.Category) error
	CreateEntry(entry *_toshl.Entry) error
}

type Client struct {
	ToshlClient toshlClient
}

func NewToshlClient(token string) (Client, error) {
	if token == "" {
		return Client{}, errors.New("toshl token is empty")
	}

	return Client{
		ToshlClient: _toshl.NewClient(token, nil),
	}, nil
}

func (c Client) client() toshlClient {
	return c.ToshlClient
}

func (c Client) GetCategories() ([]*types.Category, error) {
	categories, err := c.client().Categories(nil)
	if err != nil {
		return nil, err
	}

	var nCategories []*types.Category
	for _, category := range categories {
		cat := types.Category{
			Category: category,
		}
		nCategories = append(nCategories, &cat)
	}

	return nCategories, nil
}

func (c Client) CreateCategory(category *types.Category) error {
	if err := c.client().CreateCategory(&category.Category); err != nil {
		return err
	}
	return nil
}

func (c Client) CreateEntry(entry *types.Entry) error {
	if err := c.client().CreateEntry(&entry.Entry); err != nil {
		return err
	}
	return nil
}

func (c Client) GetAccounts() ([]*types.Account, error) {
	accounts, err := c.client().Accounts(nil)
	if err != nil {
		return nil, err
	}

	var nAccounts []*types.Account
	for _, account := range accounts {
		nAccount := &types.Account{
			Account: account,
		}
		nAccounts = append(nAccounts, nAccount)
	}

	return nAccounts, nil
}
