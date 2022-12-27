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
	"github.com/Philanthropists/toshl-email-autosync/v2/pkg/pipe"
	"github.com/pkg/errors"
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

type stackTracer interface {
	StackTrace() errors.StackTrace
}

func (s *Sync) Run(ctx context.Context) (e error) {
	defer func() {
		if err := recover(); err != nil {
			asError, ok := err.(error)
			if ok {
				e = fmt.Errorf("got panic: %w", asError)
				errStack, ok := errors.Cause(asError).(stackTracer)
				if ok {
					fmt.Printf("Stacktrace: %v\n", errStack.StackTrace())
				} else {
					s.log().Debug("error does not implement stacktracer")
				}

			} else {
				e = fmt.Errorf("got panic: %v", err)
			}
		}
	}()

	s.log().Info("running sync", zap.Bool("dryrun", s.DryRun))

	mailCl := mail.Client{
		Addr:     s.Config.Address,
		Username: s.Config.Username,
		Password: s.Config.Password,
	}
	defer func() {
		err := mailCl.Logout()
		if err != nil {
			s.log().Warn("error logging out of mail client", zap.Error(err))
		}
	}()

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

	successfulTxs := pipe.IgnoreOnError(ctx.Done(), savedTxs)

	awaitErr := s.ArchiveTransactions(ctx, &mailCl, successfulTxs)

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

	archiveErr, ok := <-awaitErr
	if ok && archiveErr != nil {
		s.log().Error("failed to archive transaction emails", zap.Error(archiveErr))
	}

	return nil
}
