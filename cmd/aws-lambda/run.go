package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"go.uber.org/zap"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
)

const (
	credentialsFile = "credentials.json"
	versionFile     = "version"
)

func getVersion() (string, error) {
	f, err := os.Open(versionFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	raw, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(raw), nil
}

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

func configureLogger() error {
	config := zap.NewProductionConfig()

	opts := []zap.Option{
		zap.AddCallerSkip(1),
	}

	logger, err := config.Build(opts...)
	if err != nil {
		return err
	}

	logging.SetCustomGlobalLogger(logger)

	return nil
}

func HandleRequest(ctx context.Context) error {
	config, err := getConfig()
	if err != nil {
		return err
	}

	if err := configureLogger(); err != nil {
		return fmt.Errorf("could not configure logger: %w", err)
	}

	sync := sync.Sync{
		Config: config,
		DryRun: false,
	}

	version := "dev"
	if v, err := getVersion(); err == nil {
		version = v
	}

	ctx = context.WithValue(ctx, types.VersionCtxKey{}, version)

	log := logging.New()
	ctx = log.With(zap.String("version", version)).GetContext(ctx)

	const awsLambdaTimeout = 140 * time.Second
	ctx, cancel := context.WithTimeout(ctx, awsLambdaTimeout)
	defer cancel()

	return sync.Run(ctx)
}

func main() {
	lambda.Start(HandleRequest)
}
