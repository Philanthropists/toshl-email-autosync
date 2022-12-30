package main

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
)

const credentialsFile = "credentials.json"

var GitCommit string

func getConfig() (types.Config, error) {
	credFile, err := os.Open(credentialsFile)
	if err != nil {
		return types.Config{}, err
	}
	defer credFile.Close()

	authBytes, err := io.ReadAll(credFile)
	if err != nil {
		return types.Config{}, err
	}

	var config types.Config
	err = json.Unmarshal(authBytes, &config)
	if err != nil {
		return types.Config{}, err
	}

	return config, nil
}

func HandleRequest(ctx context.Context) error {
	config, err := getConfig()
	if err != nil {
		return err
	}

	if GitCommit != "" && len(GitCommit) >= 3 {
		ctx = context.WithValue(ctx, types.Version, GitCommit[:3])
	}

	sync := sync.Sync {
		Config: config,
		DryRun: false,
	}

	return sync.Run(ctx)
}

func main() {
	lambda.Start(HandleRequest)
}
