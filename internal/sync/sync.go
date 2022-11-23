package sync

import (
	"context"
	"log"

	zap "go.uber.org/zap"

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
	s.log().Info("Starting to run sync", zap.Bool("dryrun", s.DryRun))

	return nil
}
