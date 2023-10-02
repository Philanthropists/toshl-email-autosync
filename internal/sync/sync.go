package sync

import (
	"context"
	"sync"
	"time"

	"github.com/zeebo/errs"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank/banktypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
)

var syncErr = errs.Class("sync")

type (
	VersionCtxKey struct{}
)

type banksRepository interface {
	GetBanks(context.Context) []banktypes.BankDelegate
}

type dateRepository interface {
	GetLastProcessedDate(context.Context) (time.Time, error)
	SaveProcessedDate(context.Context, time.Time) error
}

type Dependencies struct {
	TimeLocale *time.Location
	BanksRepo  banksRepository
	DateRepo   dateRepository
}

type Sync struct {
	Config types.Config
	DryRun bool

	configOnce sync.Once
	deps       *Dependencies
}

func (s *Sync) Run(ctx context.Context) (genErr error) {
	log := logging.New()
	defer func() { _ = log.Sync() }()
	defer func() { genErr = syncErr.Wrap(genErr) }()

	if err := s.configure(ctx); err != nil {
		return err
	}

	log.Info("timelocale set", logging.String("timezone", s.Config.Timezone))

	// Guidelines:
	// - For every client, receive a context
	// - Try to include as many fallback ops as possible
	// - Make a general repository for data that abstracts every action taken
	// - Always use minimal abstractions for dependencies, and use interfaces always
	// Always log into the general logger
	// Always include a DryRun context value
	// Always use errs library for errors

	// TODO: get available banks
	banks := s.deps.BanksRepo.GetBanks(ctx)
	_ = banks

	// TODO: get last successful transaction timestamp
	lastProcessedDate, err := s.deps.DateRepo.GetLastProcessedDate(ctx)
	if err != nil {
		return err
	}

	log.Info("last processed date", logging.Time("last_processed_date", lastProcessedDate))

	// TODO: get all mail entries from mailbox, also beware of context cancelation

	// TODO: when processing each mail, get each user config for handling notifications (use a cache aswell)

	// TODO: if there are parse errors, each should be archived into the "error parsing" mailbox

	// TODO: successful parses, are now being registered into the accounting software

	// TODO: each sucessfull to register into accounting is to be archived into the 'processed' mailbox

	// TODO: notify each user with the processing report

	return nil
}
