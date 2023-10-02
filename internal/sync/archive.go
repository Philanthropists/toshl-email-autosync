package sync

import (
	"context"

	mail "github.com/Philanthropists/toshl-email-autosync/v2/internal/mail/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/pkg/pipe"
)

type mailMoveClient interface {
	Mailboxes() ([]mail.Mailbox, error)
	Move(mail.Mailbox, ...uint32) error
}

func (s *Sync) ArchiveTransactions(
	ctx context.Context,
	mailCl mailMoveClient,
	txs <-chan *types.TransactionInfo,
) <-chan error {
	// mailbox := mail.Mailbox(s.Config.ArchiveMailbox)
	mailbox := mail.Mailbox("")
	if mailbox == "" {
		panic("archive mailbox name cannot be zero-value")
	}

	mailboxes, err := mailCl.Mailboxes()
	if err != nil {
		panic(err)
	}

	assertMailboxExists(mailboxes, mailbox)

	res := make(chan error, 1)
	go func() {
		defer close(res)

		// s.log().Debug("gathering messages ids to archive")
		txsIds := pipe.Gather(ctx.Done(), txs,
			func(t *types.TransactionInfo) (uint32, error) {
				return t.MsgId, nil
			},
		)

		var ids []uint32
		for _, id := range txsIds {
			if id.Error == nil {
				ids = append(ids, id.Value)
			}
		}

		if len(ids) == 0 {
			// s.log().Debug("no messages to archive")
			return
		}

		// s.log().Debug("moving messages to archive mailbox",
		// 	zap.String("mailbox", string(mailbox)),
		// 	zap.Reflect("ids", ids),
		// )

		if s.DryRun {
			// s.log().Info("not moving messages to archive mailbox",
			// 	zap.Bool("dryrun", s.DryRun),
			// )
			return
		}

		err := mailCl.Move(mailbox, ids...)
		if err != nil {
			res <- err
		}
	}()

	return res
}

func assertMailboxExists(mailboxes []mail.Mailbox, archiveMailbox mail.Mailbox) {
	found := false
	for _, mailbox := range mailboxes {
		if mailbox == archiveMailbox {
			found = true
			break
		}
	}

	if !found {
		panic("archive mailbox not found " + archiveMailbox)
	}
}
