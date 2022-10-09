package market

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Philanthropists/toshl-email-autosync/internal/logger"
	"github.com/Philanthropists/toshl-email-autosync/internal/market/investment-fund/bancolombia"
	bancoltypes "github.com/Philanthropists/toshl-email-autosync/internal/market/investment-fund/bancolombia/types"
	"github.com/Philanthropists/toshl-email-autosync/internal/market/rapidapi"
	rapidapitypes "github.com/Philanthropists/toshl-email-autosync/internal/market/rapidapi/types"
	"github.com/Philanthropists/toshl-email-autosync/internal/notifications"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
)

func shouldRun(ctx context.Context, locale *time.Location, times []string) bool {
	// TODO: Use DynamoDB to get the last reported date or last stock data
	const timeLayout = "15:04"
	const timeframeDifference = 7

	ct, _ := time.Parse(timeLayout, time.Now().In(locale).Format(timeLayout))

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

	if auth.TwilioAccountSid == "" {
		log.Warn("No Twilio Account SID was specified, no stocks or funds are going to be retrieved")
		return nil
	}

	err := processStocks(ctx, auth)
	if err != nil {
		return err
	}

	err = processFunds(ctx, auth)
	if err != nil {
		return err
	}

	return nil
}

func processStocks(ctx context.Context, auth types.Auth) error {
	log := logger.GetLogger()
	stockOptions := auth.StockOptions
	locale, err := time.LoadLocation(auth.Timezone)
	if err != nil {
		return err
	}

	if !stockOptions.Enabled || !shouldRun(ctx, locale, stockOptions.Times) {
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

	sendStockInformation(stocks)

	return nil
}

func processFunds(ctx context.Context, auth types.Auth) error {
	log := logger.GetLogger()
	fundOptions := auth.FundOptions
	locale, err := time.LoadLocation(auth.Timezone)
	if err != nil {
		return err
	}

	if !fundOptions.Enabled || !shouldRun(ctx, locale, fundOptions.Times) {
		log.Infow("Not getting funds information")
		return nil
	}

	fundNames := auth.FundOptions.Funds

	funds, err := getInvestmentFunds(ctx, fundNames)
	if err != nil {
		return err
	}

	sendFundsNotification(funds)

	return nil
}

func sendStockInformation(stocks map[rapidapitypes.Stock]float64) {
	const stockFmt string = "(%5s) = $%.2f USD"

	log := logger.GetLogger()

	var stockMsgs []string
	for name, value := range stocks {
		msg := fmt.Sprintf(stockFmt, name, value)
		log.Info(msg)

		stockMsgs = append(stockMsgs, msg)
	}
	stockMsgs = append(stockMsgs, "----------------------")

	msg := strings.Join(stockMsgs, "\n")

	if err := notifications.PushNotification(msg); err != nil {
		log.Errorf("error pushing notification: %v", err)
	}
}

func getStocks(ctx context.Context, key, host string, stocks []string) (map[rapidapitypes.Stock]float64, error) {
	log := logger.GetLogger()

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

func getInvestmentFunds(ctx context.Context, funds []string) ([]bancoltypes.InvestmentFund, error) {
	log := logger.GetLogger()

	availableFunds, err := bancolombia.GetAvailableInvestmentFundsBasicInfo()
	if err != nil {
		return nil, err
	}

	fundsMap := map[string]bancoltypes.InvestmentFundBasicInfo{}
	for _, fund := range availableFunds {
		fundsMap[fund.Name] = fund
	}

	fundsInfo := []bancoltypes.InvestmentFund{}
	errorsFound := []error{}
	for _, fundName := range funds {
		fund, err := getInvestmentFundByName(fundsMap, fundName)
		if err == nil {
			fundsInfo = append(fundsInfo, fund)
		} else {
			errorsFound = append(errorsFound, err)
		}
	}

	if len(errorsFound) == len(funds) {
		return nil, errors.New("no funds were able to be retrieved")
	}

	if len(errorsFound) > 0 {
		log.Errorw("Some errors were found downloading funds",
			"funds", errorsFound)
	}

	return fundsInfo, nil
}

func getInvestmentFundByName(fundsMap map[string]bancoltypes.InvestmentFundBasicInfo, fundName string) (bancoltypes.InvestmentFund, error) {
	fundBasicInfo, ok := fundsMap[fundName]
	if !ok {
		return bancoltypes.InvestmentFund{}, fmt.Errorf("[%s] not found in available funds", fundName)
	}

	fundId := fundBasicInfo.Nit
	fund, err := bancolombia.GetInvestmentFundById(fundId)
	if err != nil {
		return bancoltypes.InvestmentFund{}, err
	}

	return fund, nil
}

func sendFundsNotification(funds []bancoltypes.InvestmentFund) {
	const fundFmt string = "[%10s] = week: %.1f%%, month: %.1f%%, year: %.1f%%"

	log := logger.GetLogger()

	var fundMsgs []string
	for _, fund := range funds {
		name := fund.Name
		weeklyPercentage := fund.Profitability.Days.WeeklyPercentage
		monthlyPercentage := fund.Profitability.Days.MonthlyPercentage
		yearPercentage := fund.Profitability.Years.Current
		msg := fmt.Sprintf(fundFmt, name, weeklyPercentage, monthlyPercentage, yearPercentage)
		log.Info(msg)

		fundMsgs = append(fundMsgs, msg)
	}
	fundMsgs = append(fundMsgs, "----------------------")

	msg := strings.Join(fundMsgs, "\n")

	if err := notifications.PushNotification(msg); err != nil {
		log.Errorf("error pushing notification: %v", err)
	}
}
