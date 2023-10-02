package sync

import (
	"context"
	"fmt"
	"strings"

	stypes "github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types"
)

type notifClient interface {
	SendMessage(msg string) error
}

func (s *Sync) SendNotifications(
	ctx context.Context,
	client notifClient,
	nots <-chan types.Notification,
) <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		const headerFmt = `%s Txs: s:%d / f:%d / parse:%d`

		var (
			succ, fail, parse int64
			msgs              []string
		)

		for n := range nots {
			switch n.Type {
			case types.Success:
				succ += 1
			case types.Failed:
				fail += 1
			case types.Parse:
				parse += 1
			}

			msgs = append(msgs, n.String())
		}

		if len(msgs) == 0 {
			// s.log().Info("no messages to send, not sending any notification",
			// 	zap.Int("msgs", len(msgs)),
			// )
			return
		}

		version := ctx.Value(stypes.Version)
		if version == nil {
			version = "dev"
		}

		header := fmt.Sprintf(headerFmt, version, succ, fail, parse)

		msg := strings.Join(append([]string{header}, msgs...), "\n")

		// s.log().Info("sending notifications",
		// 	zap.Int64("success", succ),
		// 	zap.Int64("failed", fail),
		// 	zap.Int64("parse", parse),
		// )

		if s.DryRun {
			// s.log().Info("not sending notifications",
			// 	zap.Bool("dryrun", s.DryRun),
			// )
			return
		}

		err := client.SendMessage(msg)
		if err != nil {
			errCh <- err
		}
	}()

	return errCh
}
