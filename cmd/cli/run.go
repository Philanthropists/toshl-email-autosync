package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"

	zap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
)

const credentialsFile = "credentials.json"

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
	execute := flag.Bool("execute", false, "execute actual changes")
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

	err = sync.Run(context.Background())
	if err != nil {
		logger.Fatal("failed to run sync", zap.Error(err))
	}
}
