package types

import (
	_imap "github.com/emersion/go-imap"
)

type Mailbox string

type Message struct {
	*_imap.Message

	RawBody []byte
}

type Filter func(message Message) bool
