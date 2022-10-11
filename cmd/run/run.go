package main

import (
	"context"
	"encoding/json"
	"flag"
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
)

const credentialsFile = "credentials.json"

var GitCommit string

type Options struct {
	DryRun bool
	Debug  bool
}

func getOptions() Options {
	defer flag.Parse()

	var options Options

	flag.BoolVar(&options.DryRun, "dryRun", false, "Tell what will happen but not execute")
	flag.BoolVar(&options.Debug, "debug", false, "Output debug output")

	return options
}

func getAuth() (types.Auth, error) {
	credFile, err := os.Open(credentialsFile)
	if err != nil {
		return types.Auth{}, err
	}
	defer credFile.Close()

	authBytes, err := io.ReadAll(credFile)
	if err != nil {
		return types.Auth{}, err
	}

	var auth types.Auth
	err = json.Unmarshal(authBytes, &auth)
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

func main() {
	common.PrintVersion(GitCommit)
	_ = getOptions()

	auth, err := getAuth()
	if err != nil {
		panic(err)
	}

	closeFn := SetupNotifications(auth)
	defer closeFn()

	var wg concurrency.WaitGroup
	wg.Add(2)

	go func() {
		errThis := sync.Run(context.Background(), auth)
		if errThis != nil {
			err = errThis
		}
		wg.Done()
	}()

	go func() {
		errThis := market.Run(context.Background(), auth)
		if errThis != nil {
			err = errThis
		}
		wg.Done()
	}()

	wg.Wait()

	if err != nil {
		panic(err)
	}
}
