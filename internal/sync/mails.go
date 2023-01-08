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
	Messages(ctx context.Context, box mail.Mailbox, since time.Time) (<-chan pipe.Result[*mail.Message], error)
}

func (s *Sync) GetMessagesFromInbox(ctx context.Context, c mailClient, banks []types.BankDelegate, since time.Time) (<-chan pipe.Result[*types.BankMessage], error) {
	const mailbox = "INBOX"

	s.log().Info("processing messages from mailbox",
		zap.Time("since", since),
		zap.String("mailbox", mailbox),
		zap.String("mail", s.Config.Username),
	)

	msgs, err := c.Messages(ctx, mailbox, since)
	if err != nil {
		return nil, err
	}

	filteredMsgs := pipe.ConcurrentMap(ctx.Done(), s.goroutines(), msgs, func(res pipe.Result[*mail.Message]) (*types.BankMessage, error) {
		m := res.Value

		msg := types.Message{
			Message: m,
		}

		var from []string
		for _, address := range msg.Envelope.From {
			f := address.Address()
			from = append(from, f)
		}

		for _, b := range banks {
			if res.Error != nil && b.ComesFrom(from) {
				s.log().Info("msg comes from bank and has errors",
					zap.Stringer("bank", b),
					zap.Time("msgDate", m.Envelope.Date),
					zap.Error(res.Error),
				)
				return nil, res.Error
			}

			if b.FilterMessage(msg) {
				bm := &types.BankMessage{
					Message: msg,
					Bank:    b,
				}

				return bm, res.Error
			}
		}

		return nil, ErrMessageBankNotFound
	})

	return filteredMsgs, nil
}
