package sync

import (
	"context"
	"time"

	mail "github.com/Philanthropists/toshl-email-autosync/v2/internal/mail/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/pkg/pipe"
	"go.uber.org/zap"
)

type mailClient interface {
	Messages(ctx context.Context, box mail.Mailbox, since time.Time) (<-chan *mail.Message, error)
}

func (s *Sync) GetMessagesFromInbox(ctx context.Context, c mailClient, banks []types.BankDelegate, since time.Time) (<-chan pipe.Result[*types.BankMessage], error) {
	const mailbox = "INBOX"

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
