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

func (s *Sync) ArchiveSuccessfulTransactions(ctx context.Context, mailCl mailMoveClient, txs <-chan *types.TransactionInfo) <-chan pipe.Result[*types.TransactionInfo] {
	mailbox := s.Config.ArchiveMailbox
	if mailbox == "" {
		panic("archive mailbox name cannot be nil-value")
	}

	mailboxes, err := mailCl.Mailboxes()
	if err != nil {
		panic(err)
	}

	assertMailboxExists(mailboxes, mailbox)

	return pipe.Map(ctx.Done(), txs, func(t *types.TransactionInfo) (*types.TransactionInfo, error) {
		err := mailCl.Move(mail.Mailbox(mailbox), t.MsgId)

		return t, err
	})
}

func assertMailboxExists(mailboxes []mail.Mailbox, archiveMailbox string) {
	found := false
	for _, mailbox := range mailboxes {
		if string(mailbox) == archiveMailbox {
			found = true
			break
		}
	}

	if !found {
		panic("archive mailbox not found " + archiveMailbox)
	}
}
