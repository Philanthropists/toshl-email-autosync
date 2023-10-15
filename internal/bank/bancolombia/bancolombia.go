package bancolombia

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/zeebo/errs"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank/banktypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank/validation"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types/currency"
	regexp_util "github.com/Philanthropists/toshl-email-autosync/v2/internal/util/regexp"
)

var bancolombiaErr = errs.Class("bancolombia")

type Bancolombia struct{}

func (b Bancolombia) String() string {
	return "Bancolombia"
}

func (b Bancolombia) ComesFrom(from []string) bool {
	for _, f := range from {
		switch f {
		case "alertasynotificaciones@notificacionesbancolombia.com":
			fallthrough
		case "alertasynotificaciones@bancolombia.com.co":
			return true
		}
	}

	return false
}

func (b Bancolombia) FilterMessage(msg banktypes.Message) bool {
	text := string(msg.Body())
	_, keep := regexp_util.MatchesAnyRegexp(regexMatching, text)

	return keep
}

var regexMatching = []*regexp_util.Match[banktypes.TrxType]{
	{
		Regexp: regexp.MustCompile(
			`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) a (?P<place>.+) desde (?:cta|T\.CRED) \*(?P<account>\d{4})\.`,
		),
		Value: banktypes.Expense,
	},
	{
		Regexp: regexp.MustCompile(
			`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) en (?P<place>[^\.]+)\..+T\.Cred \*(?P<account>\d{4})\.`,
		),
		Value: banktypes.Expense,
	},
	{
		Regexp: regexp.MustCompile(
			`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) en (?P<place>.+)\..+T\.(?:Cred|Deb) \*(?P<account>\d{4})\.`,
		),
		Value: banktypes.Expense,
	},
	{
		Regexp: regexp.MustCompile(
			`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) desde cta \*(?P<account>\d{4}).+cta (?P<place>\d{9,16})\.`,
		),
		Value: banktypes.Expense,
	},
	{
		Regexp: regexp.MustCompile(
			`Realizaste una (?P<type>\w+) con QR por \$(?P<value>[0-9,\.]+), desde cta \*(?P<account>\d{4}) a cta (?P<place>\d{9,16})\.`,
		),
		Value: banktypes.Expense,
	},
	{
		Regexp: regexp.MustCompile(
			`Bancolombia le informa (?P<type>\w+) de pago de (?P<place>[A-Z\s]+) por \$(?P<value>[0-9,\.]+) en su cuenta (?P<account>[A-Z\s]+)\s.+\.`,
		),
		Value: banktypes.Income,
	},
	{
		Regexp: regexp.MustCompile(
			`Bancolombia te informa (?P<type>\w+) transferencia de (?P<place>[A-Z\s]+) por \$(?P<value>[0-9,\.]+) en la cuenta \*(?P<account>[0-9]+)\.`,
		),
		Value: banktypes.Income,
	},
	{
		Regexp: regexp.MustCompile(
			`Bancolombia le informa un (?P<type>\w+) (?P<place>[\w\s]+) por \$(?P<value>[0-9,\.]+) en su Cuenta (?P<account>\w+)\.`,
		),
		Value: banktypes.Income,
	},
	{
		Regexp: regexp.MustCompile(
			`Bancolombia le informa un (?P<type>[\w\s]+) de (?P<place>[\w\s\.]+) por \$(?P<value>[0-9,\.]+) en su Cuenta (?P<account>\w+)\.`,
		),
		Value: banktypes.Income,
	},
	{
		Regexp: regexp.MustCompile(
			`Bancolombia te informa (?P<type>[\w\s]+) por \$(?P<value>[0-9,\.]+) a (?P<place>[\w\s\.]+) desde producto \*(?P<account>\w+)\.`,
		),
		Value: banktypes.Expense,
	},
}

func (b Bancolombia) ExtractTransactionInfoFromMessage(
	msg banktypes.Message,
) (_ *banktypes.TrxInfo, err error) {
	defer func() {
		err = bancolombiaErr.Wrap(err)
	}()

	text := string(msg.Body())

	selectedRegexp, ok := regexp_util.MatchesAnyRegexp(regexMatching, text)
	if !ok {
		return nil, errs.New("message did not match any regexp")
	}

	result := regexp_util.ExtractFieldsWithMatch(text, selectedRegexp)

	if !validation.ContainsAllRequiredFields(result) {
		return nil, errs.New(
			"message does not contain all required fields: [result:%+v]",
			result,
		)
	}

	value, err := getValueFromText(result["value"])
	if err != nil {
		return nil, errs.Wrap(err)
	}

	action := result["type"]
	place := result["place"]
	account := result["account"]
	trxType := selectedRegexp.Value

	return &banktypes.TrxInfo{
		Date:          msg.Date(),
		Bank:          b,
		Action:        action,
		Description:   place,
		Account:       account,
		Value:         value,
		CorrelationID: banktypes.MessageID(msg.ID()),
		Type:          banktypes.TrxType(trxType),
	}, nil
}

// This would be way easier if Bancolombia had a consistent use of commas and dots inside the currency
var (
	currencyRegexp = regexp.MustCompile(
		`^(?P<integer>[0-9\.,]+)[\.,](?P<decimal>\d{2})$`,
	)
	currencyRegexpWithoutDecimal = regexp.MustCompile(`^(?P<integer>[0-9\.,]+)`)
)

func getValueFromTextWithDecimal(s string) (string, error) {
	if !currencyRegexp.MatchString(s) {
		return "", fmt.Errorf("string [%s] does not match regex [%s]", s, currencyRegexp.String())
	}

	res := regexp_util.ExtractFields(s, currencyRegexp)
	integer, ok := res["integer"]
	if !ok {
		return "", fmt.Errorf("string [%s] should have an integer part", s)
	}

	decimal, ok := res["decimal"]
	if !ok {
		return "", fmt.Errorf("string [%s] should have a decimal part", s)
	}

	integer = strings.ReplaceAll(integer, ",", "")
	integer = strings.ReplaceAll(integer, ".", "")
	valueStr := integer + "." + decimal

	return valueStr, nil
}

func getValueFromTextWithoutDecimal(s string) (string, error) {
	if !currencyRegexpWithoutDecimal.MatchString(s) {
		return "", fmt.Errorf(
			"string [%s] does not match regex without decimal [%s]",
			s,
			currencyRegexp.String(),
		)
	}

	res := regexp_util.ExtractFields(s, currencyRegexpWithoutDecimal)
	integer, ok := res["integer"]
	if !ok {
		return "", fmt.Errorf("string [%s] should have an integer part", s)
	}

	integer = strings.ReplaceAll(integer, ",", "")
	integer = strings.ReplaceAll(integer, ".", "")
	decimal := "0"
	valueStr := integer + "." + decimal

	return valueStr, nil
}

func getValueFromText(s string) (currency.Amount, error) {
	valueStr, err := getValueFromTextWithDecimal(s)
	if err != nil {
		valueStr, err = getValueFromTextWithoutDecimal(s)
	}

	if err != nil {
		return currency.Amount{}, err
	}

	value, err := strconv.ParseFloat(valueStr, 64)

	var amount currency.Amount
	amount.Code = "COP"
	amount.Number = value

	return amount, err
}
