package sync

import (
	"context"
	"fmt"
	"log"
	"runtime"

	zap "go.uber.org/zap"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/mail"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/store/saas/toshl"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
	gTypes "github.com/Philanthropists/toshl-email-autosync/v2/internal/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/pkg/pipe"
)

type Sync struct {
	Config types.Config
	DryRun bool

	Log        *zap.Logger
	Goroutines uint
}

func (s *Sync) log() *zap.Logger {
	if s.Log == nil {
		logger, err := zap.NewProduction()
		if err != nil {
			log.Panicf("could not create logger: %v", err)
		}
		s.Log = logger
	}

	return s.Log
}

func (s *Sync) goroutines() uint {
	if s.Goroutines == 0 {
		cpus := runtime.NumCPU()
		return uint(cpus)
	}

	return s.Goroutines
}

func (s *Sync) Run(ctx context.Context) (e error) {
	defer func() {
		if err := recover(); err != nil {
			e = fmt.Errorf("got panic: %v", err)
		}
	}()

	s.log().Info("running sync", zap.Bool("dryrun", s.DryRun))

	mailCl := mail.Client{
		Addr:     s.Config.Address,
		Username: s.Config.Username,
		Password: s.Config.Password,
	}
	defer mailCl.Logout()

	banks := s.AvailableBanks()

	msgs, err := s.GetMessagesFromInbox(ctx, &mailCl, banks)
	if err != nil {
		return err
	}

	matchedMsgs := pipe.IgnoreOnError(ctx.Done(), msgs)

	txs := s.ExtractTransactionInfoFromMessages(ctx, matchedMsgs)

	toshlCl, err := toshl.NewToshlClient(s.Config.Toshl.Token)
	if err != nil {
		return err
	}

	cleanTxs := pipe.IgnoreOnError(ctx.Done(), txs)

	savedTxs := s.SaveTransactions(ctx, toshlCl, cleanTxs)

	savedTxs, teeSavedTxs := pipe.Tee(ctx.Done(), savedTxs)

	successTxs := pipe.IgnoreOnError(ctx.Done(), savedTxs)

	archived := s.ArchiveSuccessfulTransactions(ctx, &mailCl, successTxs)

	out := pipe.OnError(ctx.Done(), archived, func(t *gTypes.TransactionInfo, err error) {
		if err != nil {
			s.log().Error("failed to archive transaction email", zap.Reflect("entry", *t), zap.Error(err))
		}
	})

	pipe.NopConsumer(ctx.Done(), out)

	for t := range teeSavedTxs {
		tx := t.Value
		logCtx := s.log().With(
			zap.Reflect("date", tx.Date),
			zap.String("account", tx.Account),
			zap.String("place", tx.Place),
			zap.Reflect("value", tx.Value),
			zap.String("bank", tx.Bank.String()),
		)

		if t.Error == nil {
			logCtx.Info("created entry for transaction")
		} else {
			logCtx.Error("failed to create entry for transaction", zap.Error(t.Error))
		}
	}

	return nil
}
