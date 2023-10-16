package mailserv

import (
	"context"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"
	"github.com/zeebo/errs"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/services/mailserv/mailservtypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types/result"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/util/utilslices"
)

type IMAPClient interface {
	List(ref string, name string, ch chan *imap.MailboxInfo) error
	Select(name string, readOnly bool) (*imap.MailboxStatus, error)
	UidSearch(criteria *imap.SearchCriteria) (seqNums []uint32, err error)
	UidFetch(seqset *imap.SeqSet, items []imap.FetchItem, ch chan *imap.Message) error
	// Copy(seqset *imap.SeqSet, dest string) error
	// UidStore(*imap.SeqSet, imap.StoreItem, interface{}, chan *imap.Message) error
	UidMove(seqset *imap.SeqSet, dest string) error
}

type MessageErr result.ConcreteResult[mailservtypes.Message]

type IMAPService struct {
	NewImapFunc func() IMAPClient

	poolOnce sync.Once
	clPool   sync.Pool
}

func (r *IMAPService) getClient() IMAPClient {
	r.poolOnce.Do(func() {
		f := func() any {
			return r.NewImapFunc()
		}
		r.clPool = sync.Pool{
			New: f,
		}
	})

	cl := r.clPool.Get()
	if cl == nil {
		log := logging.New()
		defer func() { _ = log.Sync() }()

		log.Panic("could not create imap client")
	}

	return cl.(IMAPClient)
}

func (r *IMAPService) GetAvailableMailboxes(
	ctx context.Context,
) ([]string, error) {
	rawMailboxes := make(chan *imap.MailboxInfo)
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		errCh <- r.getClient().List("", "*", rawMailboxes)
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

func (r *IMAPService) GetMessagesFromMailbox(
	ctx context.Context,
	mailbox string,
	since time.Time,
) (<-chan result.Result[mailservtypes.Message], error) {
	client := r.getClient()

	_, err := client.Select(mailbox, true)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	criteria := imap.NewSearchCriteria()
	criteria.Since = since
	ids, err := client.UidSearch(criteria)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	log := logging.New()
	log.Debug("got messages since a date",
		logging.Time("since", since),
		logging.String("mailbox", mailbox),
		logging.Int("len", len(ids)),
	)

	var routines int = runtime.NumCPU()
	routines = min(routines, len(ids))

	if routines == 0 {
		// no messages to process
		c := make(chan result.Result[mailservtypes.Message])
		close(c)
		return c, nil
	}

	buckets, err := utilslices.Split(routines, ids)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	ctx, cancel := context.WithCancel(ctx)

	msgs := make(chan result.Result[mailservtypes.Message], routines)

	var wg sync.WaitGroup
	wg.Add(routines)
	for i := 0; i < routines; i++ {
		go func(ctx context.Context, ids []uint32, out chan<- result.Result[mailservtypes.Message]) {
			defer wg.Done()
			msgs, err := r.getMessagesFromMailbox(ctx, mailbox, ids...)
			if err != nil {
				log.Error("could not get messages from message bucket",
					logging.Error(err),
				)
				return
			}

			for {
				select {
				case <-ctx.Done():
					return
				case m, ok := <-msgs:
					if !ok {
						return
					}
					select {
					case <-ctx.Done():
						return
					case out <- m:
					}
				}
			}
		}(
			ctx,
			buckets[i],
			msgs,
		)
	}

	go func() {
		defer cancel()
		defer close(msgs)
		wg.Wait()
	}()

	return msgs, nil
}

func (r *IMAPService) getMessagesFromMailbox(
	ctx context.Context,
	mailbox string,
	ids ...uint32,
) (<-chan result.Result[mailservtypes.Message], error) {
	client := r.getClient()

	_, err := client.Select(mailbox, true)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	messages := make(chan *imap.Message)

	errCh := make(chan error)
	go func(out chan<- error) {
		defer close(errCh)

		fetch := imap.FetchAll.Expand()
		sections := []imap.BodySectionName{
			{
				BodyPartName: imap.BodyPartName{},
				Peek:         true,
			},
		}
		for _, s := range sections {
			fetch = append(fetch, s.FetchItem())
		}

		fetchErr := client.UidFetch(seqset, fetch, messages)
		if fetchErr != nil {
			fetchErr = errs.New("failed to fetch messages: %w", fetchErr)
		}

		out <- fetchErr
	}(errCh)

	ctx, cancel := context.WithCancel(ctx)
	msgs := r.getCompleteMessages(ctx, mailbox, messages)

	go func(in <-chan error) {
		select {
		case <-ctx.Done():
			return

		case e, ok := <-in:
			if ok && e != nil {
				defer cancel()
				log := logging.New()
				defer func() { _ = log.Sync() }()

				log.Error("there was a problem fetching messages",
					logging.Error(e),
				)
			}
		}
	}(errCh)

	return msgs, nil
}

func (r *IMAPService) getCompleteMessages(
	ctx context.Context,
	mailbox string,
	msgs <-chan *imap.Message,
) <-chan result.Result[mailservtypes.Message] {
	out := make(chan result.Result[mailservtypes.Message])

	go func() {
		defer close(out)
		for {
			var (
				m  *imap.Message
				ok bool
			)
			select {
			case <-ctx.Done():
				return

			case m, ok = <-msgs:
				if !ok {
					return
				}
			}

			msg, err := r.getCompleteMessage(m)
			select {
			case <-ctx.Done():
				return
			case out <- result.ConcreteResult[mailservtypes.Message]{
				Error: err,
				Val:   msg,
			}:
			}
		}
	}()

	return out
}

func (r *IMAPService) getCompleteMessage(
	msg *imap.Message,
) (mailservtypes.Message, error) {
	var section imap.BodySectionName
	t := msg.GetBody(&section)
	if t == nil {
		return mailservtypes.Message{}, errs.New("msg has no body")
	}
	mr, err := mail.CreateReader(t)
	if err != nil && mr == nil {
		return mailservtypes.Message{}, errs.New("could not create reader: %w", err)
	}
	defer func() { _ = mr.Close() }()

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
				return mailservtypes.Message{}, errs.New("could not read from InlineHeader body: %w", err)
			}
		}
	}

	if body == nil {
		return mailservtypes.Message{}, errs.New("no body found in msg")
	}

	return mailservtypes.Message{
		Message:  *msg,
		BodyData: body,
	}, nil
}

func (r *IMAPService) MoveMessagesToMailbox(
	_ context.Context,
	fromMailbox,
	toMailbox string,
	msgIDs ...uint32,
) error {
	if len(msgIDs) == 0 {
		// no messages to move
		return nil
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(msgIDs...)

	c := r.getClient()
	_, err := c.Select(fromMailbox, false)
	if err != nil {
		return errs.Wrap(err)
	}

	err = c.UidMove(seqset, toMailbox)
	return errs.Wrap(err)
}
