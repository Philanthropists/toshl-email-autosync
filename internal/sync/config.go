package sync

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/zeebo/errs"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/repository/dateprocessingrepo"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
)

func (s *Sync) configure(ctx context.Context) error {
	var genErr error

	deps, err := getDependencies(ctx, s.Config)
	if err != nil {
		return err
	}

	s.configOnce.Do(func() {
		s.deps = deps
	})

	return genErr
}

func getDependencies(ctx context.Context, config types.Config) (*Dependencies, error) {
	loc, err := getTimezone(config.Timezone)
	if err != nil {
		return nil, err
	}

	const region = "us-east-1"
	dynamoClient, err := getDynamoDBClient(ctx, region)
	if err != nil {
		return nil, err
	}

	return &Dependencies{
		TimeLocale: loc,
		BanksRepo:  bank.Repository{},
		DateRepo: dateprocessingrepo.DateProcessingRepositoryDynamoDB{
			DynamoDBClient: dynamoClient,
		},
	}, nil
}

func getTimezone(location string) (*time.Location, error) {
	if location == "" {
		return nil, errs.New("timezone locale should not be empty")
	}

	loc, err := time.LoadLocation(location)
	return loc, errs.Wrap(err)
}

func getDynamoDBClient(ctx context.Context, region string) (*dynamodb.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, errs.Wrap(err)
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)
	return dynamoClient, nil
}
