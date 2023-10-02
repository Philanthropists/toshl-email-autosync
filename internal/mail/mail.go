package mail

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	_imap "github.com/emersion/go-imap"
	_client "github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/mail/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/pkg/pipe"
)

const (
	fetchTimeout = 10 * time.Second
)

type Client struct {
	Addr     string
	Username string
	Password string

	once       sync.Once
	imapClient *_client.Client

	locker sync.Mutex
}

func (c *Client) client() (*_client.Client, error) {
	if c.imapClient == nil {
		return c.createClient()
	}

	return c.imapClient, nil
}

func (c *Client) createClient() (*_client.Client, error) {
	emailClient, err := _client.DialTLS(c.Addr, nil)
	if err != nil {
		return nil, err
	}

	if err := emailClient.Login(c.Username, c.Password); err != nil {
		return nil, err
	}

	c.once.Do(func() {
		c.imapClient = emailClient
	})

	return emailClient, nil
}

func (c *Client) Mailboxes() ([]types.Mailbox, error) {
	c.locker.Lock()
	defer c.locker.Unlock()
	client, err := c.client()
	if err != nil {
		return nil, err
	}

	rawMailboxes := make(chan *_imap.MailboxInfo)
	done := make(chan error, 1)
	defer close(done)
	go func() {
		done <- client.List("", "*", rawMailboxes)
	}()

	var mailboxes []types.Mailbox
	for m := range rawMailboxes {
		mailbox := types.Mailbox(m.Name)
		mailboxes = append(mailboxes, mailbox)
	}

	if err := <-done; err != nil {
		return nil, err
	}

	return mailboxes, nil
}

func (c *Client) Select(box types.Mailbox) error {
	c.locker.Lock()
	defer c.locker.Unlock()
	client, err := c.client()
	if err != nil {
		return err
	}

	_, err = client.Select(string(box), true)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Messages(
	ctx context.Context,
	box types.Mailbox,
	since time.Time,
) (<-chan pipe.Result[*types.Message], error) {
	c.locker.Lock()
	defer c.locker.Unlock()
	client, err := c.client()
	if err != nil {
		return nil, err
	}

	_, err = client.Select(string(box), true)
	if err != nil {
		return nil, err
	}

	criteria := _imap.NewSearchCriteria()
	criteria.Since = since
	ids, err := client.Search(criteria)
	if err != nil {
		return nil, err
	}

	seqset := new(_imap.SeqSet)
	seqset.AddNum(ids...)

	messages := make(chan *_imap.Message)

	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)

	asyncErr := pipe.AsyncResult(ctx.Done(), func() (bool, error) {
		var section _imap.BodySectionName
		fetch := []_imap.FetchItem{section.FetchItem(), _imap.FetchEnvelope}
		err := client.Fetch(seqset, fetch, messages)
		if err != nil {
			return false, err
		}

		return true, nil
	})

	newCtx, cancel := context.WithCancel(ctx)

	go func() {
		err, ok := <-asyncErr
		if ok && err.Error != nil {
			cancel()
		}
	}()

	// Workaround since imap client is trying to send messages and it does not receive any context to cancel
	messagesPipe := pipe.StopOnClose[*_imap.Message](newCtx.Done(), messages)

	const cons = 10
	msgs := pipe.ConcurrentMap(
		newCtx.Done(),
		cons,
		messagesPipe,
		func(m *_imap.Message) (*types.Message, error) {
			msg, err := getCompleteMessage(m)
			return &msg, err
		},
	)

	return msgs, nil
}

func getCompleteMessage(_msg *_imap.Message) (types.Message, error) {
	body, err := getMessageBody(_msg)
	if err != nil {
		return types.Message{
				Message: _msg,
			}, types.ErrInternal{
				CauseErr: err,
				Msg:      "could not get message body",
			}
	}

	return types.Message{
		Message: _msg,
		RawBody: body,
	}, nil
}

func getMessageBody(_msg *_imap.Message) ([]byte, error) {
	var section _imap.BodySectionName
	t := _msg.GetBody(&section)
	mr, err := mail.CreateReader(t)
	if err != nil && mr == nil {
		return nil, fmt.Errorf("could not create reader: %w", err)
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
				return nil, fmt.Errorf("could not read from InlineHeader body: %w", err)
			}
		}
	}

	if body == nil {
		return nil, errors.New("no body found in msg")
	}

	return body, nil
}

func (c *Client) Move(dest types.Mailbox, ids ...uint32) error {
	client, err := c.client()
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	seqset := new(_imap.SeqSet)
	seqset.AddNum(ids...)

	err = client.Move(seqset, string(dest))
	if err != nil {
		log.Printf("could not move, moving with COPY, STORE, EXPUNGE: %v", err)
		return c.moveFallback(seqset, string(dest))
	}
	return err
}

func (c *Client) moveFallback(seqset *_imap.SeqSet, dest string) error {
	client, err := c.client()
	if err != nil {
		return err
	}

	if err := client.Copy(seqset, dest); err != nil {
		return fmt.Errorf("could not copy: %w", err)
	}

	item := _imap.FormatFlagsOp(_imap.AddFlags, false)
	flags := []interface{}{_imap.DeletedFlag}
	if err := client.Store(seqset, item, flags, nil); err != nil {
		return fmt.Errorf("could not store: %w", err)
	}

	if err := client.Expunge(nil); err != nil {
		return fmt.Errorf("could not expunge: %w", err)
	}

	return nil
}

func (c *Client) Logout() error {
	if c.imapClient == nil {
		return nil
	}

	if err := c.imapClient.Logout(); err != nil {
		return fmt.Errorf("could not logout: %w", err)
	}

	return nil
}
