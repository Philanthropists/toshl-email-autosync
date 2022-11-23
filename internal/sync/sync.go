package sync

import (
	"context"
	"fmt"
	"log"

	zap "go.uber.org/zap"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/mail"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
)

type Sync struct {
	Config types.Config
	DryRun bool

	Log *zap.Logger
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

	var c int32
	for m := range msgs {
		date := m.Envelope.Date
		sub := m.Envelope.Subject
		bank := m.Bank.String()
		fmt.Printf("[%v][%s][%s]\n", date, sub, bank)
		c++
	}

	s.log().Info("processed messages", zap.Int32("count", c))

	return nil
}
