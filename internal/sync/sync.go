package sync

import (
	"context"
	"errors"

	"github.com/rs/zerolog"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/entities"
)

type Sync struct {
	Config entities.Config
	DryRun bool
	Log    zerolog.Logger
}

func (s *Sync) Run(ctx context.Context) error {
	s.Log.Info().Msg("Correctly set logger")

	return errors.New("test error")
}
