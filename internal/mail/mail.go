package mail

import (
	"errors"
	"fmt"
	"io"
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

	return emailClient, nil
}

func CreateImapClient(addr, username, password string) (Client, error) {
	c := Client{
		Addr:     addr,
		Username: username,
		Password: password,
	}

	imapClient, err := c.createClient()
	if err != nil {
		return Client{}, err
	}

	c.imapClient = imapClient
	return c, nil
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

func (c *Client) Messages(box types.Mailbox, since time.Time) (<-chan *types.Message, error) {
	client, err := c.client()
	if err != nil {
		return nil, err
	}

	status, err := client.Select(string(box), true)
	if err != nil {
		return nil, err
	}

	if !status.ReadOnly {
		return nil, errors.New("mailbox should be readonly")
	}

	criteria := _imap.NewSearchCriteria()
	criteria.Since = since
	ids, err := client.Search(criteria)
	if err != nil {
		return nil, err
	}

	seqset := new(_imap.SeqSet)
	seqset.AddNum(ids...)

	const cons = 10
	messages := make(chan *_imap.Message, cons)
	done := make(chan struct{})

	var section _imap.BodySectionName
	fetch := []_imap.FetchItem{section.FetchItem(), _imap.FetchEnvelope}
	go func() {
		err := client.Fetch(seqset, fetch, messages)
		if err != nil {
			close(done)
		}
	}()

	msgs := pipe.ConcurrentMap(done, cons, messages, func(m *_imap.Message) pipe.Result[*types.Message] {
		msg, err := getCompleteMessage(m)
		return pipe.Result[*types.Message]{
			Value: &msg,
			Error: err,
		}
	})

	filteredMsgs := pipe.IgnoreOnError(done, msgs)

	return filteredMsgs, nil
}

func getCompleteMessage(_msg *_imap.Message) (types.Message, error) {
	body, err := getMessageBody(_msg)
	if err != nil {
		return types.Message{}, err
	}

	return types.Message{
		Message: _msg,
		RawBody: body,
	}, nil
}

func getMessageBody(_msg *_imap.Message) ([]byte, error) {
	var section _imap.BodySectionName
	t := _msg.GetBody(&section)
	mr, _ := mail.CreateReader(t)

	var body []byte
	for body == nil {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}

		switch p.Header.(type) {
		case *mail.InlineHeader:
			// This is the message's text (can be plain-text or HTML)
			body, _ = io.ReadAll(p.Body)
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

	return err
}

func (c *Client) Logout() error {
	client, err := c.client()
	if err != nil {
		return fmt.Errorf("could not logout: %w", err)
	}

	if err := client.Logout(); err != nil {
		return err
	}

	return nil

}
