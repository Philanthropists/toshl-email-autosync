package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/Philanthropists/toshl-go"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
)

const credentialsFile = "credentials.json"

type Config struct {
	Toshl types.Toshl
}

func getConfig() (Config, error) {
	credFile, err := os.Open(credentialsFile)
	if err != nil {
		return Config{}, err
	}
	defer credFile.Close()

	raw, err := io.ReadAll(credFile)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(raw, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func openOrCreateFile(s string) *os.File {
	file, err := os.OpenFile(s, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("could not get categories: %s", err.Error())
	}

	return file
}

func main() {
	cfg, err := getConfig()
	if err != nil {
		log.Fatalf("could not get config: %s", err.Error())
	}

	c := toshl.NewClient(cfg.Toshl.Token, nil)

	// categories
	cats, err := c.Categories(nil)
	if err != nil {
		log.Fatalf("could not get categories: %s", err.Error())
	}

	fmt.Printf("got %d categories\n", len(cats))
	catsFile := openOrCreateFile("categories.json")
	defer catsFile.Close()

	d, _ := json.Marshal(cats)
	fmt.Fprintf(catsFile, "%s\n", string(d))

	// entries
	entries, err := c.Entries(&toshl.EntryQueryParams{
		From: toshl.Date(time.Now().Add(-1 * time.Hour * 24 * 365)),
		To:   toshl.Date(time.Now()),
	})
	if err != nil {
		log.Fatalf("could not get categories: %s", err.Error())
	}

	fmt.Printf("got %d entries\n", len(entries))

	entriesFile := openOrCreateFile("entries.json")
	defer entriesFile.Close()

	d, _ = json.Marshal(entries)
	fmt.Fprintf(entriesFile, "%s\n", string(d))
}
