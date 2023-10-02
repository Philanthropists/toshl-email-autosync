package mailrepo

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"
	"github.com/zeebo/errs"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
)

type imapClient interface {
	List(ref string, name string, ch chan *imap.MailboxInfo) error
	Select(name string, readOnly bool) (*imap.MailboxStatus, error)
	Search(criteria *imap.SearchCriteria) (seqNums []uint32, err error)
	Fetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
	// Copy(seqset *imap.SeqSet, dest string) error
	Store(seqset *imap.SeqSet, item imap.StoreItem, value interface{}, ch chan *imap.Message) error
}

type Message struct {
	imap.Message
	BodyData []byte
}

func (m Message) ID() uint64 {
	return uint64(m.Message.SeqNum)
}

func (m Message) From() []string {
	addrs := m.Message.Envelope.From
	from := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		from = append(from, addr.Address())
	}

	return from
}

func (m Message) Subject() string {
	return m.Message.Envelope.Subject
}

func (m Message) Date() time.Time {
	return m.Message.Envelope.Date
}

func (m Message) Body() []byte {
	return m.BodyData
}

type MessageErr struct {
	Err error
	Msg Message
}

func (e MessageErr) Error() string {
	return fmt.Sprintf("message %d has error %v", e.Msg.SeqNum, e.Err)
}

func (e MessageErr) Unwrap() error {
	return e.Err
}

type MessageErrs struct {
	Errs []MessageErr
}

func (e MessageErrs) Error() string {
	return fmt.Sprintf("%d messages got error", len(e.Errs))
}

func (e MessageErrs) Unwrap() []error {
	errs := make([]error, 0, len(e.Errs))
	for _, e := range e.Errs {
		errs = append(errs, e)
	}
	return errs
}

type IMAPRepository struct {
	IMAPClient  imapClient
	NewImapFunc func() imapClient

	clPool sync.Pool
}

func (r *IMAPRepository) GetAvailableMailboxes(
	ctx context.Context,
) ([]string, error) {
	rawMailboxes := make(chan *imap.MailboxInfo)
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		errCh <- r.IMAPClient.List("", "*", rawMailboxes)
	}()

	var (
		mailboxes []string
		ok        bool
	)
	ok = true
	for ok {
		var m *imap.MailboxInfo

		select {
		case <-ctx.Done():
			return nil, errs.Wrap(ctx.Err())

		case err := <-errCh:
			if err != nil {
				return nil, errs.Wrap(err)
			}

		case m, ok = <-rawMailboxes:
		}

		if m != nil {
			name := m.Name
			mailboxes = append(mailboxes, name)
		}
	}

	return mailboxes, nil
}

func (r *IMAPRepository) GetMessagesFromMailbox(
	ctx context.Context,
	mailbox string,
	since time.Time,
) (<-chan MessageErr, error) {
	_, err := r.IMAPClient.Select(mailbox, true)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	criteria := imap.NewSearchCriteria()
	criteria.Since = since
	ids, err := r.IMAPClient.Search(criteria)
	if err != nil {
		return nil, err
	}

	log := logging.New()
	log.Debug("got messages since a date",
		logging.Time("since", since),
		logging.String("mailbox", mailbox),
		logging.Int("len", len(ids)),
	)

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	const bufferSize = 10
	messages := make(chan *imap.Message, bufferSize)

	errCh := make(chan error)
	go func() {
		defer close(errCh)

		var section imap.BodySectionName
		fetch := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope}
		fetchErr := r.IMAPClient.Fetch(seqset, fetch, messages)
		if fetchErr != nil {
			fetchErr = errs.New("failed to fetch messages: %w", fetchErr)
		}

		errCh <- fetchErr
	}()

	ctx, cancel := context.WithCancel(ctx)
	msgs := r.getCompleteMessages(ctx, messages)

	go func() {
		select {
		case <-ctx.Done():
			return

		case e, ok := <-errCh:
			if ok && e != nil {
				defer cancel()
				log := logging.New()
				defer func() { _ = log.Sync() }()

				log.Error("there was a problem fetching messages",
					logging.Error(e),
				)
			}
		}
	}()

	return msgs, nil
}

func (r *IMAPRepository) getCompleteMessages(
	ctx context.Context,
	msgs <-chan *imap.Message,
) <-chan MessageErr {
	const bufferSize = 40
	out := make(chan MessageErr, bufferSize)

	const n = bufferSize
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(done <-chan struct{}) {
			defer wg.Done()

			for {
				select {
				case <-done:
					return
				case m, ok := <-msgs:
					if !ok {
						return
					}

					msg, err := r.getCompleteMessage(done, m)
					select {
					case <-done:
						return
					case out <- MessageErr{
						Err: err,
						Msg: msg,
					}:
					}

				}
			}
		}(ctx.Done())
	}

	go func(ctx context.Context) {
		defer close(out)
		wg.Wait()
		if err := ctx.Err(); err != nil {
			log := logging.New()
			defer func() { _ = log.Sync() }()

			log.Warn("context ended with error", logging.Error(err))
		}
	}(ctx)

	return out
}

func (r *IMAPRepository) getCompleteMessage(
	done <-chan struct{},
	msg *imap.Message,
) (Message, error) {
	var section imap.BodySectionName
	t := msg.GetBody(&section)
	if t == nil {
		return Message{}, errs.New("msg has no body")
	}
	mr, err := mail.CreateReader(t)
	if err != nil && mr == nil {
		return Message{}, errs.New("could not create reader: %w", err)
	}

	var body []byte
	for body == nil {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}

		switch p.Header.(type) {
		case *mail.InlineHeader:
			// This is the message's text (can be plain-text or HTML)
			body, err = io.ReadAll(p.Body)
			if err != nil {
				return Message{}, errs.New("could not read from InlineHeader body: %w", err)
			}
		}
	}

	if body == nil {
		return Message{}, errs.New("no body found in msg")
	}

	return Message{
		Message:  *msg,
		BodyData: body,
	}, nil
}

func (r *IMAPRepository) MoveMessagesToMailbox(
	ctx context.Context,
	toMailbox string,
	msgIDs ...uint64,
) error {
	return nil
}
