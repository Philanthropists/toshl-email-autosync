package sync

import (
	"context"
	"fmt"
	"log"
	"runtime"

	zap "go.uber.org/zap"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/mail"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
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

func (s *Sync) Run(ctx context.Context) error {
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

	msgs, teeMsgs := pipe.Tee(ctx.Done(), msgs)

	errTxs := pipe.AwaitResult(ctx.Done(), func() (int64, error) {
		var f int64
		for m := range teeMsgs {
			if m.Error != nil {
				f++
			}
		}

		return f, nil
	})

	matchedMsgs := pipe.IgnoreOnError(ctx.Done(), msgs)

	var c int32
	for m := range matchedMsgs {
		date := m.Envelope.Date
		sub := m.Envelope.Subject
		bank := m.Bank.String()
		fmt.Printf("[%v][%s][%s]\n", date, sub, bank)
		c++
	}

	s.log().Info("processed messages", zap.Int32("count", c), zap.Int64("failures", (<-errTxs).Value))

	return nil
}
