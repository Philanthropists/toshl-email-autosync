package market

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Philanthropists/toshl-email-autosync/internal/logger"
	"github.com/Philanthropists/toshl-email-autosync/internal/market/rapidapi"
	rapidapitypes "github.com/Philanthropists/toshl-email-autosync/internal/market/rapidapi/types"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
)

//go:embed stocks.json
var stocksDat []byte

type stocksJson struct {
	Stocks []string `json:"stocks"`
}

var stocks []rapidapitypes.Stock

func init() {
	var dat stocksJson
	err := json.Unmarshal(stocksDat, &dat)
	if err != nil {
		panic("unable to parse stock list")
	}

	for _, stockName := range dat.Stocks {
		stocks = append(stocks, rapidapitypes.Stock(stockName))
	}
}

func shouldRun(ctx context.Context) bool {
	// TODO Use DynamoDB to get the last reported date or last stock data
	return false
}

func Run(ctx context.Context, auth types.Auth) error {
	log := logger.GetLogger()
	defer log.Sync()

	if !shouldRun(ctx) {
		log.Infow("Not getting stock information")
		return nil
	}

	stocks, err := getStocks(ctx, auth)
	if err != nil {
		return err
	}

	if auth.TwilioAccountSid != "" {
		sendStockInformation(auth, stocks)
	}

	return nil
}

func sendStockInformation(auth types.Auth, stocks map[rapidapitypes.Stock]float64) {
	const stockFmt string = "(%5s) = $%.2f USD"

	log := logger.GetLogger()
	defer log.Sync()

	var stockMsgs []string
	for name, value := range stocks {
		msg := fmt.Sprintf(stockFmt, name, value)
		log.Info(msg)

		stockMsgs = append(stockMsgs, msg)
	}
	stockMsgs = append(stockMsgs, "----------------------")

	msg := strings.Join(stockMsgs, "\n")

	sync.SendNotifications(auth, msg)
}

func getStocks(ctx context.Context, auth types.Auth) (map[rapidapitypes.Stock]float64, error) {
	log := logger.GetLogger()
	defer log.Sync()

	key := auth.RapidApiKey
	host := auth.RapidApiHost
	api, err := rapidapi.GetMarketClient(key, host)
	if err != nil {
		panic("unable to create rapid api client")
	}

	values := map[rapidapitypes.Stock]float64{}
	for _, stock := range stocks {
		value, err := api.GetMarketValue(stock)
		if err != nil {
			log.Errorw("error getting value for stock",
				"error", err)
			continue
		}

		values[stock] = value
	}

	if len(values) == 0 {
		return nil, errors.New("was not able to get any stock information")
	}

	return values, nil
}
