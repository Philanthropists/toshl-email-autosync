package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"time"

	zap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
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

	raw, err := io.ReadAll(credFile)
	if err != nil {
		return types.Config{}, err
	}

	var config types.Config
	err = json.Unmarshal(raw, &config)
	if err != nil {
		return types.Config{}, err
	}

	return config, nil
}

func configureLogger(execute bool) error {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	opts := []zap.Option{
		zap.AddCallerSkip(1),
	}

	logger, err := config.Build(opts...)
	if err != nil {
		return err
	}

	version := "dev"
	if GitCommit != "" && len(GitCommit) >= 3 {
		version = GitCommit[:3]
	}

	logger = logger.With(zap.String("version", version))
	if !execute {
		logger = logger.With(zap.Bool("dryrun", true))
	}
	logging.SetCustomGlobalLogger(logger)

	return nil
}

func main() {
	ctx := context.Background()

	var (
		execute bool
		timeout string
	)

	flag.BoolVar(&execute, "execute", false, "execute actual changes")
	flag.StringVar(&timeout, "timeout", "", "timeout for sync to cancel")
	flag.Parse()

	if err := configureLogger(execute); err != nil {
		log.Fatal(err)
	}

	log := logging.New()

	config, err := getConfig()
	if err != nil {
		log.Fatal("failed to get config", logging.Error(err))
	}

	sync := sync.Sync{
		Config: config,
		DryRun: !execute,
	}

	if timeout != "" {
		t, err := time.ParseDuration(timeout)
		if err != nil {
			log.Fatal("the specified duration is invalid",
				logging.String("user_input", timeout),
				logging.Error(err),
			)
		}
		nctx, cancel := context.WithTimeout(ctx, t)
		ctx = nctx
		defer cancel()
	}

	err = sync.Run(ctx)
	if err != nil {
		log.Fatal("failed to run sync", zap.Error(err))
	}
}
