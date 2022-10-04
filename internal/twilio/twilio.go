package twilio

import (
	"encoding/json"
	"errors"

	_twilio "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type Client interface {
	SendMsg(from, to, msg string) (string, error)
}

func NewClient(accountSid, authToken string) (*twilioClient, error) {
	if accountSid == "" || authToken == "" {
		return nil, errors.New("account sid and auth token cannot be empty")
	}

	client := _twilio.NewRestClientWithParams(_twilio.RestClientParams{
		Username: accountSid,
		Password: authToken,
	})

	if client == nil {
		panic("twilio client is nil, this is unexpected")
	}

	return &twilioClient{client: client}, nil
}

type twilioClient struct {
	client *_twilio.RestClient
}

func (c twilioClient) SendMsg(from, to, msg string) (string, error) {
	if from == "" || to == "" || msg == "" {
		return "", errors.New("none of the parameters can be empty")
	}

	params := &openapi.CreateMessageParams{}
	params.SetFrom(from)
	params.SetTo(to)
	params.SetBody(msg)

	message, err := c.client.ApiV2010.CreateMessage(params)
	if err != nil {
		return "", err
	}

	response, err := json.Marshal(*message)
	if err != nil {
		return "", err
	}

	return string(response), nil
}
