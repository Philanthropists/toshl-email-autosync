package twilio

import (
	"sync"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/external/twilio/twiliotypes"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"github.com/zeebo/errs"
)

var twilioErr = errs.Class("twilio")

type twilioAPIService interface {
	CreateMessage(params *twilioApi.CreateMessageParams) (*twilioApi.ApiV2010Message, error)
}

type Client struct {
	AccountSid string
	Token      string
	From       string

	once   sync.Once
	client twilioAPIService
}

func (c *Client) init() {
	c.once.Do(func() {
		c.client = twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: c.AccountSid,
			Password: c.Token,
		}).Api
	})
}

func (c *Client) SendMessage(toNumber, sms string) (_ twiliotypes.APIResponse, genErr error) {
	c.init()
	defer func() {
		genErr = twilioErr.Wrap(genErr)
	}()

	if toNumber == "" || sms == "" {
		return twiliotypes.APIResponse{}, errs.New("none of the parameters can be empty")
	}

	ps := &twilioApi.CreateMessageParams{}
	ps.SetFrom(c.From)
	ps.SetTo(toNumber)
	ps.SetBody(sms)

	message, err := c.client.CreateMessage(ps)
	if err != nil {
		return twiliotypes.APIResponse{}, errs.Wrap(err)
	}

	r := twiliotypes.APIResponse{
		ApiV2010Message: message,
	}

	return r, nil
}
