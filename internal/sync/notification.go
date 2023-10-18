package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank/banktypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
)

func (s *Sync) notifyUsers(ctx context.Context, responses []registerResponse) error {
	log := logging.FromContext(ctx)

	notifPerUser := make(map[string][]registerResponse, len(responses))
	for _, resp := range responses {
		cfg := resp.Cfg

		email := cfg.Email
		v, ok := notifPerUser[email]
		if !ok {
			v = make([]registerResponse, 0, 1)
		}

		v = append(v, resp)
		notifPerUser[email] = v
	}

	for k, v := range notifPerUser {
		smsNumber := v[0].Cfg.SMSDeliveryNumber
		err := s.notifyUserWithSMS(ctx, smsNumber, v)
		if err != nil {
			log.Error("could not set sms to user",
				logging.String("email", k),
				logging.String("smsNumber", smsNumber),
				logging.Error(err),
			)
		}
	}

	return nil
}

func (s *Sync) notifyUserWithSMS(
	ctx context.Context,
	toNumber string,
	responses []registerResponse,
) error {
	log := logging.FromContext(ctx)

	const headerFmt = `%s Se registraron %d transacciones`
	const limit = 2

	version := "dev"
	if v, ok := ctx.Value(types.VersionCtxKey{}).(string); ok {
		if len(v) > 3 {
			version = v[:3]
		}
	}

	msgs := []string{fmt.Sprintf(headerFmt, version, len(responses))}

	size := min(limit, len(responses))
	for i := 0; i < size; i++ {
		const dateFmt = "2006-01-02"
		const entryFmt = `%s %.20q %s$%.0f`

		r := responses[i]
		date := r.Trx.Date
		description := r.Trx.Description
		value := r.Trx.Value.Number
		sign := ""

		if r.Trx.Type == banktypes.Expense {
			sign = "-"
		}

		s := fmt.Sprintf(entryFmt,
			date.Format(dateFmt),
			description,
			sign,
			value,
		)

		msgs = append(msgs, s)
	}

	if len(responses) > size {
		diff := len(responses) - size
		msgs = append(msgs, fmt.Sprintf("... y otras %d", diff))
	}

	msg := strings.Join(msgs, "\n")

	if s.DryRun {
		log.Info("not sending notifications due to dryrun",
			logging.String("to_number", toNumber),
			logging.String("msg", msg),
		)
		return nil
	}

	err := s.deps.NotificationServ.SendSMS(ctx, toNumber, msg)
	return err
}
