package sync

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	zap "go.uber.org/zap"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/mail"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/notification/twilio"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/store/nosql/dynamodb"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/store/saas/toshl"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
	gTypes "github.com/Philanthropists/toshl-email-autosync/v2/internal/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/pkg/pipe"
	"github.com/pkg/errors"
)

var (
	sinceFallbackDate time.Time = time.Now().Add(-30 * 24 * 60 * 60 * time.Second) // 30 days
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

func (s *Sync) getDynamoClient(ctx context.Context) (dynamodb.Client, error) {
	const region = "us-east-1"
	dynamoClient, err := dynamodb.NewDynamoDBClient(ctx, region)
	return dynamoClient, err
}

func (s *Sync) getLastProcessedDate(ctx context.Context) time.Time {
	logCtx := s.log().With(zap.Time("fallbackDate", sinceFallbackDate))

	client, err := s.getDynamoClient(ctx)
	if err != nil {
		logCtx.Error("could not create dynamodb client",
			zap.Error(err),
		)
		return sinceFallbackDate
	}

	since, err := s.LastProcessedDate(ctx, client)
	if err != nil {
		logCtx.Error("could not get date from dynamodb", zap.Error(err))
		return sinceFallbackDate
	}

	return since
}

type notifClientWrapper struct {
	Client twilio.Client
	From   string
	To     string
}

func (n notifClientWrapper) SendMessage(msg string) error {
	_, err := n.Client.SendMessage(n.From, n.To, msg)
	return err
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

	s.log().Info("running sync",
		zap.Bool("dryrun", s.DryRun),
		zap.Reflect("version", ctx.Value(types.Version)),
	)

	mailCl := mail.Client{
		Addr:     s.Config.Address,
		Username: s.Config.Username,
		Password: s.Config.Password,
	}
	defer func() {
		err := mailCl.Logout()
		if err != nil {
			s.log().Warn("could not log out from mail client", zap.Error(err))
		}
	}()

	banks := s.AvailableBanks()
	since := s.getLastProcessedDate(ctx)
	notifCh := make(chan gTypes.Notification, 1)

	notifClient := notifClientWrapper{
		Client: twilio.Client{
			AccountSid: s.Config.Twilio.AccountSid,
			Token:      s.Config.Twilio.AuthToken,
		},
		From: s.Config.Twilio.FromNumber,
		To:   s.Config.Twilio.ToNumber,
	}

	asyncErr := s.SendNotifications(ctx, notifClient, notifCh)
	defer func(asyncErr <-chan error) {
		close(notifCh)

		err, ok := <-asyncErr
		if ok {
			s.log().Error("failed to send notifications", zap.Error(err))
			e = err
		}
	}(asyncErr)

	msgs, err := s.GetMessagesFromInbox(ctx, &mailCl, banks, since)
	if err != nil {
		return err
	}

	matchedMsgs := pipe.IgnoreOnError(ctx.Done(), msgs)

	txs := s.ExtractTransactionInfoFromMessages(ctx, matchedMsgs)

	toshlCl, err := toshl.NewToshlClient(s.Config.Toshl.Token)
	if err != nil {
		return err
	}

	cleanTxs := pipe.OnError(ctx.Done(), txs, func(tx *gTypes.TransactionInfo, err error) {
		var errParse gTypes.ErrParseFailure
		if errors.As(err, &errParse) {
			notifCh <- gTypes.Notification{
				Type: gTypes.Parse,
				Date: errParse.Message.Envelope.Date,
				Msg:  errParse.Cause.Error(),
			}
		} else {
			s.log().Error(
				"error processing message, not a ErrParseFailure type",
				zap.Error(err),
			)
		}
	})

	savedTxs := s.SaveTransactions(ctx, toshlCl, cleanTxs)
	savedTxs = pipe.Observe(ctx.Done(), savedTxs, func(r pipe.Result[*gTypes.TransactionInfo]) {
		val := r.Value
		if err := r.Error; err != nil {
			var date time.Time = time.Now()
			if r.Value != nil {
				date = val.Date
			}

			notifCh <- gTypes.Notification{
				Type: gTypes.Failed,
				Date: date,
				Msg:  err.Error(),
			}
		} else {
			notifCh <- gTypes.Notification{
				Type: gTypes.Success,
				Date: val.Date,
				Msg: fmt.Sprintf(
					"$ %.2f %s",
					val.Value.Number,
					val.Place,
				),
			}
		}
	})

	savedTxs, teeSavedTxs := pipe.Tee(ctx.Done(), savedTxs)
	teeSavedTxs, teeSavedTxs2 := pipe.Tee(ctx.Done(), teeSavedTxs)

	successfulTxs := pipe.IgnoreOnError(ctx.Done(), savedTxs)
	failedTxs := pipe.OnlyOnError(ctx.Done(), teeSavedTxs)

	asyncDate := pipe.AsyncResult(ctx.Done(), func() (time.Time, error) {
		earliestDate := time.Now().Add(-24 * time.Hour)
		for v := range failedTxs {
			t := v.Value
			date := t.Date
			if date.Before(earliestDate) {
				earliestDate = date
			}
		}

		return earliestDate, nil
	})

	asyncErr = s.ArchiveTransactions(ctx, &mailCl, successfulTxs)

	for t := range teeSavedTxs2 {
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

	archiveErr, ok := <-asyncErr
	if ok && archiveErr != nil {
		s.log().Error("failed to archive transaction emails", zap.Error(archiveErr))
	}

	if client, err := s.getDynamoClient(ctx); err == nil {
		newDate := (<-asyncDate).Value
		s.log().Info("updating last processed date", zap.Reflect("date", newDate))

		err = s.UpdateLastProcessedDate(ctx, client, newDate)
		if err != nil {
			s.log().Error("could not update last processed date in dynamo",
				zap.Error(err),
			)
		}
	} else {
		s.log().Error("could not create dynamo client",
			zap.Error(err),
		)
	}

	return nil
}
