package imap

import (
	"errors"
	"io"
	"sync"
	"time"

	"github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap/types"
	"github.com/Philanthropists/toshl-email-autosync/internal/logger"
	_imap "github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type MailClient interface {
	GetMailBoxes() ([]types.Mailbox, error)
	GetMessages(mailbox types.Mailbox, since time.Time, filter types.Filter) ([]types.Message, error)
	Move(messagesIds []uint32, destMailbox types.Mailbox) error
	Logout() error
}

func GetMailClient(addr, username, password string) (MailClient, error) {
	emailClient, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, err
	}

	if err := emailClient.Login(username, password); err != nil {
		return nil, err
	}

	return &mailClientImpl{client: emailClient}, nil
}

type mailClientImpl struct {
	client *client.Client
}

func (m mailClientImpl) GetMailBoxes() ([]types.Mailbox, error) {
	rawMailboxes := make(chan *_imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- m.client.List("", "*", rawMailboxes)
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

func (m mailClientImpl) GetMessages(mailbox types.Mailbox, since time.Time, filter types.Filter) ([]types.Message, error) {
	logger := logger.GetLogger()
	if filter == nil {
		return nil, errors.New("filter function cannot be nil")
	}

	boxStatus, err := m.client.Select(string(mailbox), true)
	if err != nil {
		return nil, err
	}

	if !boxStatus.ReadOnly {
		panic("mailbox should be readonly")
	}

	criteria := _imap.NewSearchCriteria()
	criteria.Since = since
	ids, err := m.client.Search(criteria)
	if err != nil {
		return nil, err
	}

	logger.Infow("Messages",
		"len", len(ids))

	seqset := new(_imap.SeqSet)
	seqset.AddNum(ids...)

	messages := make(chan *_imap.Message, 100)
	done := make(chan error, 1)

	var section _imap.BodySectionName
	items := []_imap.FetchItem{section.FetchItem(), _imap.FetchEnvelope}
	go func() {
		done <- m.client.Fetch(seqset, items, messages)
	}()

	filteredMsgsChan := make(chan types.Message, 50)

	go func() {
		processMultipleMessages(messages, filter, filteredMsgsChan)
	}()

	var filteredMsgs []types.Message
	for msg := range filteredMsgsChan {
		filteredMsgs = append(filteredMsgs, msg)
	}

	if err := <-done; err != nil {
		return nil, err
	}

	return filteredMsgs, nil
}

func processMultipleMessages(messages <-chan *_imap.Message, filter types.Filter, outChan chan<- types.Message) {
	const concurrentRoutines = 20

	var wg sync.WaitGroup
	wg.Add(concurrentRoutines)
	for i := 0; i < concurrentRoutines; i++ {
		go func() {
			processMessages(messages, filter, outChan)
			wg.Done()
		}()
	}

	wg.Wait()
	close(outChan)
}

func processMessages(messages <-chan *_imap.Message, filter types.Filter, outChan chan<- types.Message) {
	for _msg := range messages {
		msg, err := getCompleteMessage(_msg)
		if err != nil {
			continue
		}

		if filter(msg) {
			outChan <- msg
		}
	}
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

func (m mailClientImpl) Move(ids []uint32, destMailbox types.Mailbox) error {
	if len(ids) == 0 {
		return nil
	}

	seqset := new(_imap.SeqSet)
	seqset.AddNum(ids...)

	err := m.client.Move(seqset, string(destMailbox))

	return err
}

func (m mailClientImpl) Logout() error {
	if err := m.client.Logout(); err != nil {
		return err
	}

	return nil
}
