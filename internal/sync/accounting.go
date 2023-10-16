package sync

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/zeebo/errs"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank/banktypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/repository/accountingrepo/accountingrepotypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/repository/userconfigrepo"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types/currency"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types/result"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/util/utilregexp"
	utilslices "github.com/Philanthropists/toshl-email-autosync/v2/internal/util/utilslices"
)

type registerResponse struct {
	Trx *banktypes.TrxInfo
	Cfg userconfigrepo.UserConfig
}

func (s *Sync) registerTrxsIntoAccounting(
	ctx context.Context,
	trxs []*banktypes.TrxInfo,
) (<-chan result.Result[registerResponse], error) {
	log := logging.FromContext(ctx)

	routines := runtime.NumCPU()
	routines = min(routines, len(trxs))

	if routines == 0 {
		// no trxs to register
		c := make(chan result.Result[registerResponse])
		close(c)
		return c, nil
	}

	buckets, err := utilslices.Split(routines, trxs)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	log = log.With(
		logging.Int("routines", routines),
		logging.Int("buckets", len(buckets)),
	)

	log.Debug("executing with goroutines")
	ctx = log.GetContext(ctx)

	out := make(chan result.Result[registerResponse], routines)

	var wg sync.WaitGroup
	wg.Add(routines)
	for i := 0; i < routines; i++ {
		go func(
			ctx context.Context,
			out chan<- result.Result[registerResponse],
			trxs []*banktypes.TrxInfo,
		) {
			defer wg.Done()

			for _, t := range trxs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				r, err := s.registerSingleTrxIntoAccounting(ctx, t)
				res := &result.ConcreteResult[registerResponse]{
					Val:   r,
					Error: err,
				}

				out <- res
			}
		}(ctx, out, buckets[i])
	}

	go func() {
		defer close(out)
		wg.Wait()
	}()

	return out, nil
}

func (s *Sync) registerSingleTrxIntoAccounting(
	ctx context.Context,
	trx *banktypes.TrxInfo,
) (registerResponse, error) {
	log := logging.FromContext(ctx)
	zeroVal := registerResponse{
		Trx: trx,
	}

	repo := s.deps.AccountingRepo

	cfg, err := s.getUserConfigFromCandidates(ctx, trx.OriginMessage.To())
	if err != nil {
		return zeroVal, err
	}
	zeroVal.Cfg = cfg

	accounts, err := repo.GetAccounts(ctx, cfg.Toshl.Token)
	if err != nil {
		return zeroVal, err
	}

	const categoryPrefix = "PENDING_"

	categoryType := trx.Type.String()
	categoryName := categoryPrefix + strings.ToUpper(categoryType)
	categoryID, err := s.createCategoryIfAbsent(ctx, cfg.Toshl.Token, categoryType, categoryName)
	if err != nil {
		return zeroVal, err
	}

	log = log.With(
		logging.String("category_id", categoryID),
		logging.String("category_type", categoryType),
		logging.String("category_name", categoryName),
	)

	accountMappings := getAccountsMapping(accounts, cfg, trx.Bank.String())

	// log.Debug("account mappings",
	// 	logging.Any("mappings", accountMappings),
	// )

	amount := trx.Value.Number
	if trx.Type == banktypes.Expense {
		amount *= -1
	}

	account, ok := accountMappings[trx.Account]
	if !ok {
		return zeroVal, errs.New("transaction does not have an assigned account %q", trx.Account)
	}

	entryInput := accountingrepotypes.CreateEntryInput{
		Date: trx.Date.In(s.deps.TimeLocale),
		Currency: currency.Amount{
			Code:   "COP",
			Number: amount,
		},
		Description: fmt.Sprintf("** %s de %s", trx.Action, trx.Description),
		AccountID:   account.ID,
		CategoryID:  categoryID,
	}

	log.Debug("entry to be created",
		logging.Any("entry", entryInput),
	)

	if s.DryRun {
		log.Info("not creating entry due to dryrun")
		return zeroVal, nil
	}

	err = repo.CreateEntry(ctx, cfg.Toshl.Token, entryInput)
	return zeroVal, err
}

func getAccountsMapping(
	accounts []accountingrepotypes.Account,
	cfg userconfigrepo.UserConfig,
	bank string,
) map[string]accountingrepotypes.Account {
	exp := regexp.MustCompile(`^(?P<accounts>[0-9\s]+) `)

	mapping := make(map[string]accountingrepotypes.Account)
	for _, a := range accounts {
		name := a.Name
		r := utilregexp.ExtractFields(name, exp)

		if acNums, ok := r["accounts"]; ok {
			nums := strings.Split(acNums, " ")
			for _, n := range nums {
				mapping[n] = a
			}
		}
	}

	userBankMappings, ok := cfg.Mapping[bank]
	if ok {
		for k, v := range userBankMappings {
			if destAccount, ok := mapping[v]; ok {
				mapping[k] = destAccount
			}
		}
	}

	return mapping
}

func (s *Sync) createCategoryIfAbsent(
	ctx context.Context,
	token, catType, category string,
) (string, error) {
	log := logging.FromContext(ctx)

	repo := s.deps.AccountingRepo

	categories, err := repo.GetCategories(ctx, token)
	if err != nil {
		return "", errs.Wrap(err)
	}

	for _, c := range categories {
		if c.Name == category {
			if c.Type != catType {
				log.Warn("category types mismatch",
					logging.String("actual", c.Type),
					logging.String("expected", catType),
				)
			}
			return c.ID, nil
		}
	}

	if s.DryRun {
		log.Info("not creating categories because of dryrun")
		return "", nil
	}

	r, err := repo.CreateCategory(ctx, token, catType, category)
	if err != nil {
		return "", errs.Wrap(err)
	}

	return r.ID, nil
}

func (s *Sync) getUserConfigFromCandidates(
	ctx context.Context,
	candidates []string,
) (userconfigrepo.UserConfig, error) {
	cfgRepo := s.deps.UserCfgRepo

	found := false
	var userCfg userconfigrepo.UserConfig
	for _, c := range candidates {
		cfg, err := cfgRepo.GetUserConfigFromEmail(ctx, c)
		if err != nil && ctx.Err() != nil {
			return userconfigrepo.UserConfig{}, ctx.Err()
		}

		if err == nil {
			userCfg = cfg
			found = true
		}
	}

	if !found {
		return userconfigrepo.UserConfig{}, errs.New("could not find user config from candidates")
	}

	return userCfg, nil
}
