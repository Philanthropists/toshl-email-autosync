package proxy

import (
	"sync"
	"time"

	"github.com/Philanthropists/toshl-go"
	"github.com/patrickmn/go-cache"
)

type ToshlClient interface {
	Categories(params *toshl.CategoryQueryParams) ([]toshl.Category, error)
	Accounts(params *toshl.AccountQueryParams) ([]toshl.Account, error)
	CreateCategory(category *toshl.Category) error
	CreateEntry(entry *toshl.Entry) error
}

type inMemoryCache interface {
	SetDefault(k string, v any)
	Get(k string) (any, bool)
	Delete(k string)
}

type ToshlCacheClient struct {
	Client          ToshlClient
	ExpirationTime  time.Duration
	CleanupInterval time.Duration

	once  sync.Once
	cache inMemoryCache
}

func (c *ToshlCacheClient) init() {
	c.once.Do(func() {
		const (
			defaultExpirationTime  = 5 * time.Minute
			defaultCleanupInterval = 1 * time.Minute
		)

		expTime := defaultExpirationTime
		if c.ExpirationTime != 0 {
			expTime = c.ExpirationTime
		}

		cleanupInt := defaultCleanupInterval
		if c.CleanupInterval != 0 {
			cleanupInt = c.CleanupInterval
		}

		c.cache = cache.New(expTime, cleanupInt)
	})
}

func (c *ToshlCacheClient) Categories(params *toshl.CategoryQueryParams) ([]toshl.Category, error) {
	c.init()
	const k = "categories"
	if v, found := c.cache.Get(k); found {
		return v.([]toshl.Category), nil
	}

	v, err := c.Client.Categories(params)
	if err != nil {
		return nil, err
	}

	c.cache.SetDefault(k, v)
	return v, nil
}

func (c *ToshlCacheClient) Accounts(params *toshl.AccountQueryParams) ([]toshl.Account, error) {
	c.init()
	const k = "accounts"
	if v, found := c.cache.Get(k); found {
		return v.([]toshl.Account), nil
	}

	v, err := c.Client.Accounts(params)
	if err != nil {
		return nil, err
	}

	c.cache.SetDefault(k, v)
	return v, nil
}

func (c *ToshlCacheClient) CreateCategory(category *toshl.Category) error {
	c.init()
	const k = "categories"
	defer c.cache.Delete(k)

	return c.Client.CreateCategory(category)
}

func (c *ToshlCacheClient) CreateEntry(entry *toshl.Entry) error {
	c.init()

	return c.Client.CreateEntry(entry)
}
