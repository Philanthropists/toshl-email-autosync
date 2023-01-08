package mail

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/mail/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/pkg/pipe"
	_imap "github.com/emersion/go-imap"
	_client "github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type Client struct {
	Addr     string
	Username string
	Password string

	once       sync.Once
	imapClient *_client.Client
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
	client, err := c.client()
	if err != nil {
		return nil, err
	}

	rawMailboxes := make(chan *_imap.MailboxInfo, 10)
	done := make(chan error)
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

func (c *Client) Messages(ctx context.Context, box types.Mailbox, since time.Time) (<-chan pipe.Result[*types.Message], error) {
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
	go func() {
		_, ok := <-newCtx.Done()
		if !ok && newCtx.Err() != nil {
			for range messages {
				// consume until finished
			}
		}
	}()

	const cons = 10
	msgs := pipe.ConcurrentMap(newCtx.Done(), cons, messages, func(m *_imap.Message) (*types.Message, error) {
		msg, err := getCompleteMessage(m)
		return &msg, err
	})

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

	return client.Move(seqset, string(dest))
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
