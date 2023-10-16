package twiliotypes

import twilioApi "github.com/twilio/twilio-go/rest/api/v2010"

type APIResponse struct {
	*twilioApi.ApiV2010Message
}
