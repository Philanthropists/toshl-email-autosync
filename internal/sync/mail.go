package sync

import (
	"github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap"
	"github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap/types"
	synctypes "github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
)

func GetEmailFromInbox(mailClient imap.MailClient, banks []synctypes.BankDelegate) ([]synctypes.BankMessage, error) {
	const inboxMailbox = "INBOX"

	since := GetLastProcessedDate()

	var messages []synctypes.BankMessage

	for _, bank := range banks {
		msgs, err := mailClient.GetMessages(inboxMailbox, since, bank.FilterMessage)
		if err != nil {
			return nil, err
		}

		for _, msg := range msgs {
			bankMsg := synctypes.BankMessage{
				Message: msg,
				Bank:    bank,
			}

			messages = append(messages, bankMsg)
		}
	}

	return messages, nil
}

func ArchiveEmailsFromSuccessfulTransactions(mailClient imap.MailClient, archiveMailbox string, successfulTransactions []*synctypes.TransactionInfo) {
	if archiveMailbox == "" {
		panic("archive mailbox name cannot be nil-value")
	}

	mailboxes, err := mailClient.GetMailBoxes()
	if err != nil {
		panic(err)
	}

	assertMailboxExists(mailboxes, archiveMailbox)

	var msgsIds []uint32
	for _, t := range successfulTransactions {
		msgsIds = append(msgsIds, t.MsgId)
	}

	err = mailClient.Move(msgsIds, types.Mailbox(archiveMailbox))
	if err != nil {
		panic(err)
	}
}

func assertMailboxExists(mailboxes []types.Mailbox, archiveMailbox string) {
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
