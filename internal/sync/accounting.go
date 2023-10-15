package sync

import (
	"context"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/zeebo/errs"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank/banktypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/repository/accountingrepo/accountingrepotypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/repository/userconfigrepo"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types/result"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/util/utilregexp"
	utilslices "github.com/Philanthropists/toshl-email-autosync/v2/internal/util/utilslices"
)

func (s *Sync) registerTrxsIntoAccounting(
	ctx context.Context,
	trxs []*banktypes.TrxInfo,
) (<-chan result.Result[*banktypes.TrxInfo], error) {
	log := logging.FromContext(ctx)

	routines := runtime.NumCPU()
	routines = min(routines, len(trxs))

	if routines == 0 {
		// no trxs to register
		c := make(chan result.Result[*banktypes.TrxInfo])
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

	out := make(chan result.Result[*banktypes.TrxInfo], routines)

	var wg sync.WaitGroup
	wg.Add(routines)
	for i := 0; i < routines; i++ {
		go func(
			ctx context.Context,
			out chan<- result.Result[*banktypes.TrxInfo],
			trxs []*banktypes.TrxInfo,
		) {
			defer wg.Done()

			for _, t := range trxs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				err := s.registerSingleTrxIntoAccounting(ctx, t)
				res := &result.ConcreteResult[*banktypes.TrxInfo]{
					Val:   t,
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

func (s *Sync) registerSingleTrxIntoAccounting(ctx context.Context, trx *banktypes.TrxInfo) error {
	log := logging.FromContext(ctx)

	repo := s.deps.AccountingRepo

	cfg, err := s.getUserConfigFromCandidates(ctx, trx.OriginMessage.To())
	if err != nil {
		return err
	}

	now := time.Now()
	accounts, err := repo.GetAccounts(ctx, cfg.Toshl.Token)
	if err != nil {
		return err
	}

	const categoryPrefix = "PENDING_"

	categoryType := trx.Type.String()
	categoryName := categoryPrefix + strings.ToUpper(categoryType)
	categoryID, err := s.createCategoryIfAbsent(ctx, cfg.Toshl.Token, categoryType, categoryName)
	if err != nil {
		return err
	}

	log.Debug("toshl registered",
		logging.Duration("took", time.Since(now)),
		logging.Any("accounts", accounts),
		logging.String("category_id", categoryID),
		logging.String("category_type", categoryType),
		logging.String("category_name", categoryName),
		logging.Any("user_cfg", cfg),
	)

	accountMappings := getAccountsMapping(accounts, cfg, trx.Bank.String())

	log.Debug("account mappings",
		logging.Any("mappings", accountMappings),
	)

	return nil
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
				log.Warn("categories mismatch",
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
