package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/external/twilio"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/sync/types"
)

const credentialsFile = "credentials.json"

type Config struct {
	types.Twilio
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

func main() {
	msg := flag.String("msg", "Test message", "Message to send")
	to := flag.String("to", "", "To number to send message")
	flag.Parse()

	config, err := getConfig()
	if err != nil {
		panic(err)
	}

	client := twilio.Client{
		AccountSid: config.Twilio.AccountSid,
		Token:      config.Twilio.AuthToken,
		From:       config.FromNumber,
	}

	var toNumber string = "" // config.ToNumber
	if to != nil && *to != "" {
		toNumber = *to
	}

	fmt.Printf("Sending from number %s to %s: %s\n", config.FromNumber, toNumber, *msg)

	res, err := client.SendMessage(toNumber, *msg)
	if err != nil {
		panic(err)
	}

	fmt.Printf("response: %v\n", res)
}
