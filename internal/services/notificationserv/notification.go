package notificationserv

import (
	"context"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/external/twilio/twiliotypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
)

type smsClient interface {
	SendMessage(toNumber, sms string) (twiliotypes.APIResponse, error)
}

type NotificationService struct {
	SMSClient smsClient
}

func (n *NotificationService) SendSMS(ctx context.Context, to, msg string) error {
	log := logging.FromContext(ctx)

	r, err := n.SMSClient.SendMessage(to, msg)
	if err != nil {
		log.Error("failed to send SMS",
			logging.Error(err),
			logging.String("to_number", to),
			logging.Int("msg_len", len(msg)),
		)
		return err
	}

	log.Debug("response from sending SMS", logging.Any("response", r))

	return err
}

func (n *NotificationService) SendMail(ctx context.Context, email string) error {
	log := logging.FromContext(ctx)

	log.DPanic("method is to be implemented")
	return nil
}
