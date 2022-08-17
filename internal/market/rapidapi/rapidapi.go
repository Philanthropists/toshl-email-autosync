package rapidapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/Philanthropists/toshl-email-autosync/internal/market/rapidapi/types"
)

const StockSummaryURI = "https://apidojo-yahoo-finance-v1.p.rapidapi.com/stock/v2/get-summary?region=US&symbol=%s"

type StockMarket interface {
	GetMarketValue(stock types.Stock) (float64, error)
}

func GetMarketClient(api, host string) (StockMarket, error) {
	if api == "" || host == "" {
		return nil, errors.New("invalid api or host")
	}

	impl := &rapidApiImpl{}
	impl.Uri = StockSummaryURI
	impl.Header.Key = api
	impl.Header.Host = host

	return impl, nil
}

type rapidApiImpl struct {
	Uri    string
	Header struct {
		Key  string `json:"key"`
		Host string `json:"host"`
	}
}

type stockValue struct {
	Price struct {
		RegularMarketOpen struct {
			Raw float64 `json:"raw"`
		} `json:"regularMarketOpen"`
	} `json:"price"`
}

func (api *rapidApiImpl) GetCredentialsFromFile(filename string) error {
	b, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("unable to read API keys from file: %w", err)
	}

	err = json.Unmarshal(b, &api.Header)
	if err != nil {
		return err
	}

	return nil
}

func (api *rapidApiImpl) GetMarketValue(symbol types.Stock) (float64, error) {
	const HeaderKey = "x-rapidapi-key"
	const HeaderHost = "x-rapidapi-host"

	if api.Uri == "" {
		api.Uri = StockSummaryURI
	}

	url := fmt.Sprintf(api.Uri, symbol)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Add(HeaderKey, api.Header.Key)
	req.Header.Add(HeaderHost, api.Header.Host)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	var value stockValue
	err = json.NewDecoder(res.Body).Decode(&value)
	if err != nil {
		return 0, err
	}

	return value.Price.RegularMarketOpen.Raw, nil
}
