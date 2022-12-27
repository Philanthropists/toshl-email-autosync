package sync

import (
	"context"
	"time"

	mail "github.com/Philanthropists/toshl-email-autosync/v2/internal/mail/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/store/nosql/dynamodb"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/pkg/pipe"
	"go.uber.org/zap"
)

func (s *Sync) LastProcessedDate(ctx context.Context) time.Time {
	var fallbackDate time.Time = time.Now().Add(-30 * 24 * 60 * 60 * time.Second)
	const region = "us-east-1"

	logCtx := s.log().With(zap.Time("fallbackDate", fallbackDate))

	client, err := dynamodb.NewDynamoDBClient(ctx, region)
	if err != nil {
		logCtx.Error("could not create dynamodb client",
			zap.Error(err),
		)
		return fallbackDate
	}

	const (
		table  = "toshl-data"
		itemId = 1
	)

	res, err := client.GetItem(ctx, table, map[string]interface{}{
		"Id": itemId,
	})
	if err != nil {
		logCtx.Error("could not get item from dynamodb table",
			zap.String("table", table),
			zap.Int("itemId", itemId),
		)
		return fallbackDate
	}

	const field = "LastProcessedDate"

	item, ok := res[field]
	if !ok {
		logCtx.Error("item does not contain required field",
			zap.String("field", field),
		)
		return fallbackDate
	}

	dateStr, ok := item.(string)
	if !ok {
		logCtx.Error("item field is not a string",
			zap.Reflect("itemField", item),
		)
		return fallbackDate
	}

	selectedDate, err := time.Parse(time.RFC822Z, dateStr)
	if err != nil {
		logCtx.Error("item field is not a string representing a date",
			zap.Reflect("itemField", dateStr),
		)
		return fallbackDate
	}

	return selectedDate
}

type mailClient interface {
	Messages(ctx context.Context, box mail.Mailbox, since time.Time) (<-chan *mail.Message, error)
}

func (s *Sync) GetMessagesFromInbox(ctx context.Context, c mailClient, banks []types.BankDelegate) (<-chan pipe.Result[*types.BankMessage], error) {
	const mailbox = "INBOX"

	since := s.LastProcessedDate(ctx)

	s.log().Info("processing messages from mailbox",
		zap.Reflect("since", since),
		zap.String("mailbox", mailbox),
	)

	msgs, err := c.Messages(ctx, mailbox, since)
	if err != nil {
		return nil, err
	}

	filteredMsgs := pipe.ConcurrentMap(ctx.Done(), s.goroutines(), msgs, func(m *mail.Message) (*types.BankMessage, error) {
		msg := types.Message{
			Message: m,
		}

		for _, b := range banks {
			if b.FilterMessage(msg) {
				bm := &types.BankMessage{
					Message: msg,
					Bank:    b,
				}

				return bm, nil
			}
		}

		return nil, ErrMessageBankNotFound
	})

	return filteredMsgs, nil
}
