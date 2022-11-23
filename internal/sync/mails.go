package sync

import (
	"context"
	"time"

	mail "github.com/Philanthropists/toshl-email-autosync/v2/internal/mail/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/pkg/pipe"
)

func (s *Sync) LastProcessedDate() time.Time {
	// TODO: implement
	return time.Now().Add(-30 * 24 * 60 * 60 * time.Second)
}

type mailClient interface {
	Messages(box mail.Mailbox, since time.Time) (<-chan *mail.Message, error)
}

func (s *Sync) GetMessagesFromInbox(ctx context.Context, c mailClient, banks []types.BankDelegate) (<-chan *types.BankMessage, error) {
	const inbox = "INBOX"

	msgs, err := c.Messages(inbox, s.LastProcessedDate())
	if err != nil {
		return nil, err
	}

	filteredMsgs := pipe.ConcurrentMap(ctx.Done(), 10, msgs, func(m *mail.Message) pipe.Result[*types.BankMessage] {
		msg := types.Message{
			Message: m,
		}

		for _, b := range banks {
			if b.FilterMessage(msg) {
				bm := &types.BankMessage{
					Message: msg,
					Bank:    b,
				}

				return pipe.Result[*types.BankMessage]{
					Value: bm,
				}
			}
		}

		return pipe.Result[*types.BankMessage]{
			Error: ErrMessageBankNotFound,
		}
	})

	matchMsgs := pipe.IgnoreOnError(ctx.Done(), filteredMsgs)

	return matchMsgs, nil
}
