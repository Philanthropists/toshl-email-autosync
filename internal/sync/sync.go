package sync

import (
	"context"
	"sync"
	"time"

	"slices"

	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank/banktypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/repository/mailrepo"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/repository/userconfigrepo"
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

type mailRepository interface {
	GetAvailableMailboxes(context.Context) ([]string, error)
	GetMessagesFromMailbox(
		context.Context, string, time.Time,
	) (<-chan mailrepo.MessageErr, error)
	MoveMessagesToMailbox(context.Context, string, ...uint64) error
}

type userConfigRepository interface {
	GetUserConfigFromEmail(context.Context, string) (userconfigrepo.UserConfig, error)
}

type Dependencies struct {
	TimeLocale  *time.Location
	BanksRepo   banksRepository
	DateRepo    dateRepository
	MailRepo    mailRepository
	UserCfgRepo userConfigRepository
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

	// TODO: get mailboxes
	mailboxes, err := s.deps.MailRepo.GetAvailableMailboxes(ctx)
	if err != nil {
		return err
	}
	log.Info("mailboxes", zap.Strings("mailboxes", mailboxes))

	if !slices.Contains(mailboxes, "INBOX") {
		return errs.New("there is no INBOX mailbox")
	}

	mailCtx := ctx
	if deadline, ok := ctx.Deadline(); ok {
		// a small portion of the time should be to get the messages,
		// if we are unable to get more messages it does not matter,
		// we should go with what we have
		remainingTime := -1 * time.Since(deadline)
		const timeFactor = 0.4
		timeSlot := time.Duration(timeFactor * float64(remainingTime))

		log.Info("we have a deadline, setting a time slot for messages",
			logging.Time("deadline", deadline),
			logging.Duration("timeslot", timeSlot),
			logging.Float("time_factor", timeFactor),
		)

		var cancel context.CancelFunc
		mailCtx, cancel = context.WithTimeout(ctx, timeSlot)
		defer cancel()
	}

	// TODO: get all mail entries from mailbox, also beware of context cancelation
	messages, err := s.deps.MailRepo.GetMessagesFromMailbox(mailCtx, "INBOX", lastProcessedDate)
	if err != nil {
		return err
	}

	// i := 0
	// for me := range messages {
	// 	m := me.Msg
	// 	fmt.Printf("%d: %v -- %s (%d bytes)\n", m.ID(), m.Subject(), m.Date(), len(m.Body()))
	// 	i++
	// }
	//
	// log.Info("got messages", logging.Int("len", i))

	// TODO: when processing each mail, get each user config for handling notifications (use a cache aswell)
	// for me := range messages {
	// 	if me.Err != nil {
	// 		continue
	// 	}
	//
	// 	log.Info("message",
	// 		logging.Strings("from", me.Msg.From()),
	// 		logging.Strings("to", me.Msg.To()),
	// 		logging.Strings("items", me.Msg.Items()),
	// 		logging.Strings("flags", me.Msg.Flags()),
	// 	)
	// }

	// other thing
	var (
		fetchFailedMsgs int = 0
		parseFailedMsgs int = 0
		totalMsgs       int = 0
		// successMsgs     int = 0

		trxs []*banktypes.TrxInfo
	)
	for me := range messages {
		if me.Err != nil {
			fetchFailedMsgs++
			continue
		}

		msg := me.Msg

		for _, bank := range banks {
			if bank.FilterMessage(msg) {
				trx, err := bank.ExtractTransactionInfoFromMessage(msg)
				if err != nil {
					parseFailedMsgs++
					break
				}

				trxs = append(trxs, trx)
			}
		}

		totalMsgs++
	}

	log.Info("message fetching status",
		logging.Int("failed", fetchFailedMsgs),
		logging.Int("parse_failed", parseFailedMsgs),
		logging.Int("total", totalMsgs),
	)

	log.Debug("transactions that we got",
		logging.Int("len_trxs", len(trxs)),
		logging.Any("trxs", trxs),
	)

	// TODO: if there are parse errors, each should be archived into the "error parsing" mailbox

	// TODO: successful parses, are now being registered into the accounting software

	// TODO: each sucessfull to register into accounting is to be archived into the 'processed' mailbox

	// TODO: notify each user with the processing report

	return nil
}
