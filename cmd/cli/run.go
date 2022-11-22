package main

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/rs/zerolog"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/entities"
)

const credentialsFile = "credentials.json"

func getConfig() (entities.Config, error) {
	credFile, err := os.Open(credentialsFile)
	if err != nil {
		return entities.Config{}, err
	}
	defer credFile.Close()

	raw, err := io.ReadAll(credFile)
	if err != nil {
		return entities.Config{}, err
	}

	var config entities.Config
	err = json.Unmarshal(raw, &config)
	if err != nil {
		return entities.Config{}, err
	}

	return config, nil
}

func main() {
	log := zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	config, err := getConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get credentials")
	}

	sync := sync.Sync{
		Config: config,
		Log:    log,
	}

	err = sync.Run(context.Background())
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
}
