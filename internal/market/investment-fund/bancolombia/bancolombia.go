package bancolombia

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Philanthropists/toshl-email-autosync/internal/market/investment-fund/bancolombia/types"
)

const (
	host          = "https://valores.grupobancolombia.com"
	fundsListPath = "consultarFondosInversion/rest/servicio/consultarListaFondos"
	fundById      = "consultarFondosInversion/rest/servicio/buscarInformacionFondo"
)

func getFormedURIWithPath(path string) string {
	return fmt.Sprintf("%s/%s", host, path)
}

func doGetRequest(url string) ([]byte, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func GetAvailableInvestmentFundsBasicInfo() ([]types.InvestmentFundBasicInfo, error) {
	url := getFormedURIWithPath(fundsListPath)
	body, err := doGetRequest(url)
	if err != nil {
		return nil, err
	}

	var funds []types.InvestmentFundBasicInfo
	err = json.Unmarshal(body, &funds)
	if err != nil {
		return nil, err
	}

	return funds, nil
}

func GetInvestmentFundById(fundId types.InvestmentFundId) (types.InvestmentFund, error) {
	url := getFormedURIWithPath(fundById) + "/" + string(fundId)
	body, err := doGetRequest(url)
	if err != nil {
		return types.InvestmentFund{}, err
	}

	var fund types.InvestmentFund
	err = json.Unmarshal(body, &fund)
	if err != nil {
		return types.InvestmentFund{}, err
	}

	return fund, nil
}
