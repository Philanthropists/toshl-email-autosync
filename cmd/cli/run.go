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

func getLogger() (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	return config.Build()
}

func main() {
	ctx := context.Background()
	if GitCommit != "" && len(GitCommit) >= 3 {
		ctx = context.WithValue(ctx, types.Version, GitCommit[:3])
	} else {
		ctx = context.WithValue(ctx, types.Version, "dev")
	}

	execute := flag.Bool("execute", false, "execute actual changes")
	timeout := flag.Uint("timeout", 0, "timeout for sync to cancel")
	flag.Parse()

	logger, err := getLogger()
	if err != nil {
		log.Panicf("could not create logger: %v", err)
	}

	config, err := getConfig()
	if err != nil {
		logger.Fatal("failed to get credentials", zap.Error(err))
	}

	sync := sync.Sync{
		Config: config,
		DryRun: !*execute,
		Log:    logger,
	}

	if *timeout != 0 {
		t := time.Duration(*timeout) * time.Second
		nctx, cancel := context.WithTimeout(ctx, t)
		ctx = nctx
		defer cancel()
	}

	err = sync.Run(ctx)
	if err != nil {
		logger.Fatal("failed to run sync", zap.Error(err))
	}
}
