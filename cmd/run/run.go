package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	concurrency "sync"

	"github.com/Philanthropists/toshl-email-autosync/internal/market"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/common"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
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

func main() {
	common.PrintVersion(GitCommit)
	_ = getOptions()

	auth, err := getAuth()
	if err != nil {
		panic(err)
	}

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
