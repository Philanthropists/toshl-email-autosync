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
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/external/twilio"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/services/accountingserv"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/services/accountingserv/proxy"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/services/dateprocessingserv"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/services/mailserv"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/services/notificationserv"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/services/userconfigserv"
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
	newImapClientFunc := func() mailserv.IMAPClient {
		log := logging.New()
		defer func() { _ = log.Sync() }()

		cl, err := getEmailClient(addr, user, pass)
		if err != nil {
			log.Error("could not create imap client", logging.Error(err))

			return nil
		}
		return cl
	}

	newToshlClientFunc := func(t string) accountingserv.ToshlClient {
		c := toshl.NewClient(t, nil)
		proxy := &proxy.ToshlCacheClient{
			Client: c,
		}
		return proxy
	}

	return &Dependencies{
		TimeLocale: loc,
		BanksRepo:  bank.Repository{},
		DateRepo: dateprocessingserv.DynamoDBService{
			Client: dynamoClient,
		},
		MailRepo: &mailserv.IMAPService{
			NewImapFunc: newImapClientFunc,
		},
		UserCfgRepo: &userconfigserv.DynamoDBService{
			Client: dynamoClient,
		},
		AccountingRepo: &accountingserv.ToshlService{
			ClientBuilder: newToshlClientFunc,
		},
		NotificationServ: &notificationserv.NotificationService{
			SMSClient: &twilio.Client{
				AccountSid: config.Twilio.AccountSid,
				Token:      config.Twilio.AuthToken,
				From:       config.Twilio.FromNumber,
			},
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
