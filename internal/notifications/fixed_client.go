package notifications

import (
	"fmt"

	"github.com/Philanthropists/toshl-email-autosync/internal/twilio"
)

type fixedClient struct {
	Client twilio.Client
	From   string
	To     string
}

func CreateFixedClient(client twilio.Client, from, to string) (NotificationsClient, error) {
	if client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}

	if from == "" || to == "" {
		return nil, fmt.Errorf("from or to locations cannot be zero-values")
	}

	return &fixedClient{
		Client: client,
		From:   from,
		To:     to,
	}, nil
}

func (c fixedClient) SendMsg(msg string) (string, error) {
	return c.Client.SendMsg(c.From, c.To, msg)
}
