package mailrepo

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-message/mail"
	"github.com/zeebo/errs"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
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

type Message struct {
	imap.Message
	BodyData []byte
}

func (m Message) ID() uint32 {
	return m.Message.SeqNum
}

func (m Message) From() []string {
	addrs := m.Message.Envelope.From
	from := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		from = append(from, addr.Address())
	}

	return from
}

func (m Message) To() []string {
	addrs := m.Message.Envelope.To
	to := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		to = append(to, addr.Address())
	}

	return to
}

func (m Message) Senders() []string {
	addrs := m.Message.Envelope.Sender
	senders := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		senders = append(senders, addr.Address())
	}

	return senders
}

func (m Message) ReplyTo() []string {
	addrs := m.Message.Envelope.ReplyTo
	senders := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		senders = append(senders, addr.Address())
	}

	return senders
}

func (m Message) Items() []string {
	its := make([]string, 0, len(m.Message.Items))
	for k := range m.Message.Items {
		its = append(its, string(k))
	}
	return its
}

func (m Message) Flags() []string {
	return m.Message.Flags
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
	NewImapFunc func() IMAPClient

	poolOnce sync.Once
	clPool   sync.Pool
}

func (r *IMAPRepository) getClient() IMAPClient {
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

func (r *IMAPRepository) GetAvailableMailboxes(
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

func (r *IMAPRepository) GetMessagesFromMailbox(
	ctx context.Context,
	mailbox string,
	since time.Time,
) (<-chan MessageErr, error) {
	client := r.getClient()

	_, err := client.Select(mailbox, true)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	criteria := imap.NewSearchCriteria()
	criteria.Since = since
	ids, err := client.UidSearch(criteria)
	if err != nil {
		return nil, err
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
		c := make(chan MessageErr)
		close(c)
		return c, nil
	}

	buckets, err := divideSlice(routines, ids)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	ctx, cancel := context.WithCancel(ctx)

	msgs := make(chan MessageErr, routines)

	var wg sync.WaitGroup
	wg.Add(routines)
	for i := 0; i < routines; i++ {
		go func(ctx context.Context, ids []uint32, out chan<- MessageErr) {
			defer wg.Done()
			msgs, err := r.getMessagesFromMailbox(ctx, mailbox, ids...)
			if err != nil {
				log.Error("could not get messages from message bucket",
					logging.Error(err),
				)
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
		}(ctx, buckets[i], msgs)
	}

	go func() {
		defer cancel()
		defer close(msgs)
		wg.Wait()
	}()

	return msgs, nil
}

func (r *IMAPRepository) getMessagesFromMailbox(
	ctx context.Context,
	mailbox string,
	ids ...uint32,
) (<-chan MessageErr, error) {
	client := r.getClient()

	_, err := client.Select(mailbox, true)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	messages := make(chan *imap.Message)

	errCh := make(chan error)
	go func() {
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

		errCh <- fetchErr
	}()

	ctx, cancel := context.WithCancel(ctx)
	msgs := r.getCompleteMessages(ctx, mailbox, messages)

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

func divideSlice[T any](n int, v []T) ([][]T, error) {
	if n <= 0 {
		return nil, errs.New("n:%d must be greater than zero", n)
	}

	div := make(map[int][]T, n)
	for i, val := range v {
		idx := i % n
		l := div[idx]
		l = append(l, val)
		div[idx] = l
	}

	var b [][]T
	for _, v := range div {
		b = append(b, v)
	}

	return b, nil
}

func (r *IMAPRepository) getCompleteMessages(
	ctx context.Context,
	mailbox string,
	msgs <-chan *imap.Message,
) <-chan MessageErr {
	out := make(chan MessageErr)

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
			case out <- MessageErr{
				Err: err,
				Msg: msg,
			}:
			}
		}
	}()

	return out
}

func (r *IMAPRepository) getCompleteMessage(
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
	msgIDs ...uint32,
) error {
	if len(msgIDs) == 0 {
		// no messages to move
		return nil
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(msgIDs...)

	c := r.getClient()
	err := c.UidMove(seqset, toMailbox)

	return errs.Wrap(err)
}
