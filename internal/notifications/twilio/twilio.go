package twilio

import (
	"encoding/json"
	"errors"

	_twilio "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type Client struct {
	AccountSid string
	Token      string

	tc *_twilio.RestClient
}

func (c *Client) client() *_twilio.RestClient {
	if c.tc == nil {
		c.tc = _twilio.NewRestClientWithParams(_twilio.RestClientParams{
			Username: c.AccountSid,
			Password: c.Token,
		})
	}

	return c.tc
}

func (c *Client) SendMsg(from, to, msg string) (string, error) {
	tc := c.client()

	if from == "" || to == "" || msg == "" {
		return "", errors.New("none of the parameters can be empty")
	}

	params := &openapi.CreateMessageParams{}
	params.SetFrom(from)
	params.SetTo(to)
	params.SetBody(msg)

	message, err := tc.ApiV2010.CreateMessage(params)
	if err != nil {
		return "", err
	}

	response, err := json.Marshal(*message)
	if err != nil {
		return "", err
	}

	return string(response), nil
}
