package sync

import (
	"context"
	"time"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank/banktypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
)

func (s *Sync) saveLastExecutionDate(ctx context.Context, msgs []banktypes.Message) error {
	log := logging.FromContext(ctx)

	earliest := time.Now()

	for _, m := range msgs {
		t := m.Date()
		if t.Before(earliest) {
			earliest = t
		}
	}

	log.Info("setting new last execution date",
		logging.Time("earliest", earliest),
	)

	if s.DryRun {
		log.Info("not changing last execution date because of dryrun")
		return nil
	}

	return s.deps.DateRepo.SaveProcessedDate(ctx, earliest)
}
