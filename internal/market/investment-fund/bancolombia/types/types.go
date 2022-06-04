package types

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/currency"
)

const (
	copUnitCode = "COP"
)

var CopUnit currency.Unit

func init() {
	var err error
	CopUnit, err = currency.ParseISO(copUnitCode)
	if err != nil {
		panic(err)
	}
}

type Money currency.Amount

func (v *Money) UnmarshalJSON(b []byte) error {
	cleaned := strings.Trim(string(b), "\"$")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return err
	}

	*v = Money(CopUnit.Amount(value))
	return nil
}

func (v Money) String() string {
	return fmt.Sprintf("%s", currency.Amount(v))
}

type Percentage float64

func (v *Percentage) UnmarshalJSON(b []byte) error {
	cleaned := strings.Trim(string(b), "\"%")
	cleaned = strings.ReplaceAll(cleaned, ",", ".")
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return err
	}

	*v = Percentage(value)
	return nil
}

type Date time.Time

func (v *Date) UnmarshalJSON(b []byte) error {
	const dateFormat = "20060102"

	cleaned := strings.Trim(string(b), "\"")
	timeDate, err := time.Parse(dateFormat, cleaned)
	if err != nil {
		return err
	}
	*v = Date(timeDate)
	return nil
}

func (v Date) String() string {
	return time.Time(v).String()
}

type InvestmentFundId string

type InvestmentFundBasicInfo struct {
	Nit  InvestmentFundId `json:"nit"`
	Name string           `json:"nombre"`
}

type InvestmentFund struct {
	InvestmentFundBasicInfo
	Score         string `json:"calificacion"`
	Term          string `json:"plazo"`
	UnitValue     Money  `json:"valorDeUnidad"`
	CurrentValue  Money  `json:"valorEnPesos"`
	Profitability struct {
		Days struct {
			WeeklyPercentage   Percentage `json:"semanal"`
			MonthlyPercentage  Percentage `json:"mensual"`
			SemesterPercentage Percentage `json:"semestral"`
		} `json:"dias"`
		Years struct {
			Current        Percentage `json:"anioCorrido"`
			LastYear       Percentage `json:"ultimoAnio"`
			LastTwoYears   Percentage `json:"ultimos2Anios"`
			LastThreeYears Percentage `json:"ultimos3Anios"`
		} `json:"anios"`
	} `json:"rentabilidad"`
	ClosingDate   Date   `json:"fechaCierre"`
	Administrator string `json:"sociedadAdministradora"`
}
