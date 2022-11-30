package sync

import (
	"context"
	"time"

	mail "github.com/Philanthropists/toshl-email-autosync/v2/internal/mail/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/pkg/pipe"
	"go.uber.org/zap"
)

func (s *Sync) LastProcessedDate() time.Time {
	// TODO: implement
	return time.Now().Add(-30 * 24 * 60 * 60 * time.Second)
}

type mailClient interface {
	Messages(ctx context.Context, box mail.Mailbox, since time.Time) (<-chan *mail.Message, error)
}

func (s *Sync) GetMessagesFromInbox(ctx context.Context, c mailClient, banks []types.BankDelegate) (<-chan pipe.Result[*types.BankMessage], error) {
	const inbox = "INBOX"

	since := s.LastProcessedDate()

	s.log().Info("processing messages from "+inbox, zap.Reflect("since", since))

	msgs, err := c.Messages(ctx, inbox, since)
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
