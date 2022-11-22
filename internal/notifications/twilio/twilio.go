package twilio

import (
	"encoding/json"
	"errors"

	_twilio "github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

type Client struct {
	AccountSid string
	Token      string

	tc *_twilio.RestClient
}

func (c *Client) client() *_twilio.RestClient {
	if c.tc == nil {
		c.tc = _twilio.NewRestClientWithParams(_twilio.ClientParams{
			Username: c.AccountSid,
			Password: c.Token,
		})
	}

	return c.tc
}

func (c *Client) SendMessage(from, to, msg string) ([]byte, error) {
	tc := c.client()

	if from == "" || to == "" || msg == "" {
		return nil, errors.New("none of the parameters can be empty")
	}

	ps := &twilioApi.CreateMessageParams{}
	ps.SetFrom(from)
	ps.SetTo(to)
	ps.SetBody(msg)

	message, err := tc.Api.CreateMessage(ps)
	if err != nil {
		return nil, err
	}

	response, err := json.Marshal(*message)
	if err != nil {
		return nil, err
	}

	return response, nil
}
