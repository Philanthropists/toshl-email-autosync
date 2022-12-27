package sync

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

const (
	table  = "toshl-data"
	field  = "LastProcessedDate"
	itemId = 1
)

type itemNosqlClient interface {
	GetItem(context.Context, string, map[string]interface{}) (map[string]interface{}, error)
}

func (s *Sync) LastProcessedDate(ctx context.Context, client itemNosqlClient) (time.Time, error) {
	res, err := client.GetItem(ctx, table, map[string]interface{}{
		"Id": itemId,
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("could not get item with id [%d] from dynamodb table [%s]: %w",
			itemId, table, err,
		)
	}

	item, ok := res[field]
	if !ok {
		return time.Time{}, fmt.Errorf("item does not contain required field [%s]: %w",
			field, err,
		)
	}

	dateStr, ok := item.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("item field is not a string: %v", item)
	}

	selectedDate, err := time.Parse(time.RFC822Z, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("item field [%s] is not a string representing a date: %w", dateStr, err)
	}

	return selectedDate, nil
}

type updateNosqlClient interface {
	UpdateItem(ctx context.Context, s string, k, m map[string]interface{}, e string) error
}

func (s *Sync) UpdateLastProcessedDate(ctx context.Context, client updateNosqlClient, date time.Time) error {
	if s.DryRun {
		s.log().Info("not updating processing date", zap.Bool("dryrun", s.DryRun))
		return nil
	}

	exp := fmt.Sprintf("set %s = :r", field)
	value := date.Format(time.RFC822Z)

	err := client.UpdateItem(ctx, table, map[string]interface{}{
		"Id": itemId,
	}, map[string]interface{}{
		":r": value,
	}, exp)

	return err
}
