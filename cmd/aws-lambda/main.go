package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	concurrency "sync"

	"github.com/Philanthropists/toshl-email-autosync/internal/logger"
	"github.com/Philanthropists/toshl-email-autosync/internal/market"
	"github.com/Philanthropists/toshl-email-autosync/internal/notifications"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/common"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
	"github.com/Philanthropists/toshl-email-autosync/internal/twilio"
	"github.com/aws/aws-lambda-go/lambda"
)

const credentialsFile = "credentials.json"

var GitCommit string

func getAuth(rawAuth []byte) (types.Auth, error) {
	auth := types.Auth{}

	err := json.Unmarshal(rawAuth, &auth)
	if err != nil {
		return types.Auth{}, err
	}

	return auth, nil
}

func CreateNotificationsClient(auth types.Auth) (notifications.NotificationsClient, error) {
	log := logger.GetLogger()

	accountSid := auth.TwilioAccountSid
	authToken := auth.TwilioAuthToken
	fromNumber := auth.TwilioFromNumber
	toNumber := auth.TwilioToNumber

	twilioClient, err := twilio.NewClient(accountSid, authToken)
	if err != nil {
		log.Errorw("could not instantiate twilio client",
			"error", err)
		return nil, err
	}

	return notifications.CreateFixedClient(twilioClient, fromNumber, toNumber)
}

func SetupNotifications(auth types.Auth) func() {
	log := logger.GetLogger()

	notifClient, err := CreateNotificationsClient(auth)
	if err != nil {
		log.Errorf("could not create notifications client: %v", err)
	}

	if err := notifications.SetNotificationsClient(notifClient); err != nil {
		log.Errorf("could not set notifications client: %s", err)
	}

	return func() {
		notifications.Close()
	}
}

func HandleRequest(ctx context.Context) error {
	log := logger.GetLogger()
	defer log.Sync()

	common.PrintVersion(GitCommit)

	credFile, err := os.Open(credentialsFile)
	if err != nil {
		return err
	}

	authBytes, err := io.ReadAll(credFile)
	if err != nil {
		return err
	}

	credFile.Close()

	auth, err := getAuth(authBytes)
	if err != nil {
		return err
	}

	closeFn := SetupNotifications(auth)
	defer closeFn()

	var wg concurrency.WaitGroup
	wg.Add(2)

	go func() {
		errThis := sync.Run(ctx, auth)
		if errThis != nil {
			err = errThis
		}
		wg.Done()
	}()

	go func() {
		errThis := market.Run(ctx, auth)
		if errThis != nil {
			err = errThis
		}
		wg.Done()
	}()

	wg.Wait()

	return err
}

func main() {
	lambda.Start(HandleRequest)
}
