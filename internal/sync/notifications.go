package sync

import (
	"github.com/Philanthropists/toshl-email-autosync/internal/logger"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
	"github.com/Philanthropists/toshl-email-autosync/internal/twilio"
)

func SendNotifications(auth types.Auth, msg string) {
	log := logger.GetLogger()

	accountSid := auth.TwilioAccountSid
	authToken := auth.TwilioAuthToken
	fromNumber := auth.TwilioFromNumber
	toNumber := auth.TwilioToNumber

	client, err := twilio.NewClient(accountSid, authToken)
	if err != nil {
		log.Errorw("could not instantiate twilio client",
			"error", err)
		return
	}

	_, err = client.SendMsg(fromNumber, toNumber, msg)
	if err != nil {
		log.Errorw("an error ocurred when sending notification sms",
			"error", err)
		return
	}
}
