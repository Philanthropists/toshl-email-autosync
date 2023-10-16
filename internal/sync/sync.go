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
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/services/accountingserv/accountingservtypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/services/mailserv/mailservtypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/services/userconfigserv"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types/result"
)

var syncErr = errs.Class("sync")

type banksService interface {
	GetBanks(context.Context) []banktypes.BankDelegate
}

type dateService interface {
	GetLastProcessedDate(context.Context) (time.Time, error)
	SaveProcessedDate(context.Context, time.Time) error
}

type mailService interface {
	GetAvailableMailboxes(context.Context) ([]string, error)
	GetMessagesFromMailbox(
		context.Context, string, time.Time,
	) (<-chan result.Result[mailservtypes.Message], error)
	MoveMessagesToMailbox(_ context.Context, _, _ string, _ ...uint32) error
}

type userConfigService interface {
	GetUserConfigFromEmail(context.Context, string) (userconfigserv.UserConfig, error)
}

type accountingService interface {
	GetAccounts(ctx context.Context, token string) ([]accountingservtypes.Account, error)
	GetCategories(ctx context.Context, token string) ([]accountingservtypes.Category, error)
	CreateCategory(
		ctx context.Context,
		token, catType, category string,
	) (accountingservtypes.Category, error)
	CreateEntry(
		ctx context.Context,
		token string,
		entryInput accountingservtypes.CreateEntryInput,
	) error
}

type notificationService interface {
	SendSMS(ctx context.Context, to, msg string) error
	SendMail(ctx context.Context, email string) error
}

type Dependencies struct {
	TimeLocale       *time.Location
	BanksRepo        banksService
	DateRepo         dateService
	MailRepo         mailService
	UserCfgRepo      userConfigService
	AccountingRepo   accountingService
	NotificationServ notificationService
}

type Sync struct {
	Config types.Config
	DryRun bool

	configOnce sync.Once
	deps       *Dependencies
}

func (s *Sync) mailSanityCheck(ctx context.Context) error {
	log := logging.New()

	mailboxes, err := s.deps.MailRepo.GetAvailableMailboxes(ctx)
	if err != nil {
		return err
	}
	log.Debug("mailboxes", zap.Strings("mailboxes", mailboxes))

	const inboxMailbox = "INBOX"
	if !slices.Contains(mailboxes, inboxMailbox) {
		return errs.New("there is no inbox mailbox")
	}

	if !slices.Contains(mailboxes, s.Config.ParseErrorMailbox) {
		return errs.New("there is no parse error mailbox: %s", s.Config.ParseErrorMailbox)
	}

	if !slices.Contains(mailboxes, s.Config.SuccessMailbox) {
		return errs.New("there is no success mailbox: %s", s.Config.SuccessMailbox)
	}

	return nil
}

func (s *Sync) Run(ctx context.Context) (genErr error) {
	log := logging.New()
	defer func() { _ = log.Sync() }()
	defer func() { genErr = syncErr.Wrap(genErr) }()

	if err := s.configure(ctx); err != nil {
		return err
	}

	log.Debug("timelocale set", logging.String("timezone", s.Config.Timezone))

	// Guidelines:
	// - For every client, receive a context
	// - Try to include as many fallback ops as possible
	// - Make a general repository for data that abstracts every action taken
	// - Always use minimal abstractions for dependencies, and use interfaces always
	// Always log into the general logger
	// Always include a DryRun context value -- **Maybe not**
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
	if err = s.mailSanityCheck(ctx); err != nil {
		return err
	}

	mailCtx := ctx
	if deadline, ok := ctx.Deadline(); ok {
		// a small portion of the time should be to get the messages,
		// if we are unable to get more messages it does not matter,
		// we should go with what we have
		remainingTime := -1 * time.Since(deadline)
		const timeFactor = 0.4
		timeSlot := time.Duration(timeFactor * float64(remainingTime))

		log.Info("we have a deadline, setting a time slot for fetching messages",
			logging.Time("deadline", deadline),
			logging.Duration("timeslot", timeSlot),
			logging.Float("time_factor", timeFactor),
		)

		var cancel context.CancelFunc
		mailCtx, cancel = context.WithTimeout(ctx, timeSlot)
		defer cancel()
	}

	// TODO: get all mail entries from mailbox, also beware of context cancelation
	messages, err := s.deps.MailRepo.GetMessagesFromMailbox(
		mailCtx,
		"INBOX",
		lastProcessedDate,
	)
	if err != nil {
		return err
	}

	// TODO: when processing each mail, get each user config for handling notifications (use a cache aswell)
	var (
		fetchFailedMsgs int = 0
		totalMsgs       int = 0
		// successMsgs     int = 0
		parseFailedMsgs []banktypes.Message

		trxs []*banktypes.TrxInfo
	)
	for me := range messages {
		if me.Err() != nil {
			fetchFailedMsgs++
			continue
		}

		msg := me.Value()

		for _, bank := range banks {
			if bank.ComesFrom(msg.From()) && bank.FilterMessage(msg) {
				trx, extractErr := bank.ExtractTransactionInfoFromMessage(msg)
				if extractErr != nil {
					parseFailedMsgs = append(parseFailedMsgs, msg)
					break
				}

				trxs = append(trxs, trx)
			}
		}
	}

	log.Debug("message fetching status",
		logging.Int("failed", fetchFailedMsgs),
		logging.Int("parse_failed", len(parseFailedMsgs)),
		logging.Int("total", totalMsgs),
	)

	log.Debug("transactions that we got",
		logging.Int("len_trxs", len(trxs)),
	)

	// TODO: if there are parse errors, each should be archived into the "error parsing" mailbox
	if moveErr := s.moveFailedToParseMessages(ctx, parseFailedMsgs); moveErr != nil {
		return moveErr
	}

	// TODO: successful parses, are now being registered into the accounting software
	processedTrxs, err := s.registerTrxsIntoAccounting(ctx, trxs)
	if err != nil {
		return err
	}

	// TODO: each sucessfull to register into accounting is to be archived into the 'processed' mailbox
	var (
		registries  []registerResponse
		successMsgs []banktypes.Message
		failedMsgs  []banktypes.Message
	)
	for t := range processedTrxs {
		v := t.Value()
		if t.Err() == nil {
			registries = append(registries, v)
			successMsgs = append(successMsgs, v.Trx.OriginMessage)
		} else {
			failedMsgs = append(failedMsgs, v.Trx.OriginMessage)
		}
	}
	moveErr := s.moveSuccessfulMessages(ctx, successMsgs)
	if moveErr != nil {
		log.Error("could not move successful messages to success mailbox",
			logging.Error(moveErr),
		)
	}
	log.Info("moved successful messages",
		logging.Int("msgs", len(successMsgs)),
	)

	// TODO: save last execution date
	if saveErr := s.saveLastExecutionDate(ctx, failedMsgs); saveErr != nil {
		log.Error("could not save last execution date", logging.Error(saveErr))
	}

	// TODO: notify each user with the processing report
	if notifErr := s.notifyUsers(ctx, registries); notifErr != nil {
		log.Error("could not notify users", logging.Error(notifErr))
	}

	return nil
}

func (s *Sync) moveFailedToParseMessages(ctx context.Context, msgs []banktypes.Message) error {
	log := logging.FromContext(ctx)

	if s.DryRun {
		log.Info("not moving failed parse messages",
			logging.Int("len_msgs", len(msgs)),
		)
		return nil
	}

	if len(msgs) == 0 {
		log.Debug("no failed parse messages to move")
		return nil
	}

	ids := make([]uint32, 0, len(msgs))
	for _, msg := range msgs {
		ids = append(ids, msg.ID())
	}

	err := s.deps.MailRepo.MoveMessagesToMailbox(ctx, "INBOX", s.Config.ParseErrorMailbox, ids...)
	if err != nil {
		return errs.New(
			"could not move mails that had parse errors to designated mailbox %q: %w",
			s.Config.ParseErrorMailbox,
			err,
		)
	}

	return nil
}

func (s *Sync) moveSuccessfulMessages(ctx context.Context, msgs []banktypes.Message) error {
	log := logging.FromContext(ctx)

	if s.DryRun {
		log.Info("not moving successful messages because of dryrun",
			logging.Int("len_msgs", len(msgs)),
		)
		return nil
	}

	if len(msgs) == 0 {
		log.Debug("no successful messages to move")
		return nil
	}

	ids := make([]uint32, 0, len(msgs))
	for _, msg := range msgs {
		ids = append(ids, msg.ID())
	}

	err := s.deps.MailRepo.MoveMessagesToMailbox(ctx, "INBOX", s.Config.SuccessMailbox, ids...)
	if err != nil {
		return errs.New(
			"could not move mails that had parse errors to designated mailbox %q: %w",
			s.Config.ParseErrorMailbox,
			err,
		)
	}

	return nil
}
