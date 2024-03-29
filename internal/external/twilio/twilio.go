package twilio

import (
	"sync"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"github.com/zeebo/errs"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/external/twilio/twiliotypes"
)

var twilioErr = errs.Class("twilio")

type Client struct {
	AccountSid string
	Token      string
	From       string

	once   sync.Once
	client *twilio.RestClient
}

func (c *Client) init() {
	c.once.Do(func() {
		c.client = twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: c.AccountSid,
			Password: c.Token,
		})
	})
}

func (c *Client) SendMessage(toNumber, sms string) (_ twiliotypes.APIResponse, genErr error) {
	// reference: https://www.twilio.com/docs/glossary/what-sms-character-limit
	const TwilioSMSRecommendedLimit = 153
	if len(sms) > TwilioSMSRecommendedLimit {
		return twiliotypes.APIResponse{}, errs.New(
			"message is larger than twilio's recommended limit [%d], it is %d long",
			TwilioSMSRecommendedLimit, len(sms),
		)
	}

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

	message, err := c.client.Api.CreateMessage(ps)
	if err != nil {
		return twiliotypes.APIResponse{}, errs.Wrap(err)
	}

	r := twiliotypes.APIResponse{
		ApiV2010Message: message,
	}

	return r, nil
}
