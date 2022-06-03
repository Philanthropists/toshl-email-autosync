package market

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Philanthropists/toshl-email-autosync/internal/logger"
	"github.com/Philanthropists/toshl-email-autosync/internal/market/rapidapi"
	rapidapitypes "github.com/Philanthropists/toshl-email-autosync/internal/market/rapidapi/types"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
)

func shouldRun(ctx context.Context, times []string) bool {
	// TODO: Use DynamoDB to get the last reported date or last stock data
	const timeLayout = "15:04"
	const timeframeDifference = 7

	ct, _ := time.Parse(timeLayout, time.Now().Format(timeLayout))

	for _, timeFrame := range times {
		fmt.Println("time", times)
		from, _ := time.Parse(timeLayout, timeFrame)
		to := from.Add(timeframeDifference * time.Minute)

		if inTimeSpan(from, to, ct) {
			return true
		}
	}

	return false
}

func inTimeSpan(start, end, check time.Time) bool {
	if start.Before(end) {
		return !check.Before(start) && !check.After(end)
	}
	if start.Equal(end) {
		return check.Equal(start)
	}
	return !start.After(check) || !end.Before(check)
}

func Run(ctx context.Context, auth types.Auth) error {
	log := logger.GetLogger()
	defer log.Sync()

	stockOptions := auth.StockOptions

	if !stockOptions.Enabled || !shouldRun(ctx, stockOptions.Times) {
		log.Infow("Not getting stock information")
		return nil
	}

	key := auth.RapidApiKey
	host := auth.RapidApiHost
	stocksNames := auth.StockOptions.Stocks

	stocks, err := getStocks(ctx, key, host, stocksNames)
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

func getStocks(ctx context.Context, key, host string, stocks []string) (map[rapidapitypes.Stock]float64, error) {
	log := logger.GetLogger()
	defer log.Sync()

	api, err := rapidapi.GetMarketClient(key, host)
	if err != nil {
		panic("unable to create rapid api client")
	}

	values := map[rapidapitypes.Stock]float64{}
	for _, stockName := range stocks {
		stock := rapidapitypes.Stock(stockName)
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
