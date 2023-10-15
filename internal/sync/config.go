package sync

import (
	"context"
	"time"

	"github.com/Philanthropists/toshl-go"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/emersion/go-imap/client"
	"github.com/zeebo/errs"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/repository/accountingrepo"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/repository/accountingrepo/proxy"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/repository/dateprocessingrepo"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/repository/mailrepo"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/repository/userconfigrepo"
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

	addr, user, pass := config.Mail.Address, config.Mail.Username, config.Mail.Password
	newImapClientFunc := func() mailrepo.IMAPClient {
		log := logging.New()
		defer func() { _ = log.Sync() }()

		cl, err := getEmailClient(addr, user, pass)
		if err != nil {
			log.Error("could not create imap client", logging.Error(err))

			return nil
		}
		return cl
	}

	newToshlClientFunc := func(t string) accountingrepo.ToshlClient {
		c := toshl.NewClient(t, nil)
		proxy := &proxy.ToshlCacheClient{
			Client: c,
		}
		return proxy
	}

	return &Dependencies{
		TimeLocale: loc,
		BanksRepo:  bank.Repository{},
		DateRepo: dateprocessingrepo.DynamoDBRepository{
			Client: dynamoClient,
		},
		MailRepo: &mailrepo.IMAPRepository{
			NewImapFunc: newImapClientFunc,
		},
		UserCfgRepo: &userconfigrepo.DynamoDBRepository{
			Client: dynamoClient,
		},
		AccountingRepo: &accountingrepo.ToshlRepository{
			ClientBuilder: newToshlClientFunc,
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

func getEmailClient(addr, username, password string) (*client.Client, error) {
	emailClient, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	if err := emailClient.Login(username, password); err != nil {
		return nil, errs.Wrap(err)
	}

	return emailClient, nil
}
