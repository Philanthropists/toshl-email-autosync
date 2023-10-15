package mailrepotypes

import (
	"fmt"
	"time"

	"github.com/emersion/go-imap"
)

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

func (m Message) MarshalText() ([]byte, error) {
	s := fmt.Sprintf("%d - %s", m.ID(), m.Subject())
	return []byte(s), nil
}
