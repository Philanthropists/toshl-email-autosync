package bancolombia

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	imaptypes "github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap/types"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/common"
	synctypes "github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
)

type Bancolombia struct{}

func (b Bancolombia) String() string {
	return "Bancolombia"
}

func (b Bancolombia) FilterMessage(msg imaptypes.Message) bool {
	keep := true
	keep = keep && msg.Message != nil
	keep = keep && msg.Message.Envelope != nil

	if keep {
		keep = false
		for _, address := range msg.Message.Envelope.From {
			from := address.Address()
			if from == "alertasynotificaciones@notificacionesbancolombia.com" {
				keep = true
				break
			}

			if from == "alertasynotificaciones@bancolombia.com.co" {
				keep = true
				break
			}
		}
	}

	if keep {
		text := string(msg.RawBody)
		keep, _ = common.GenericMatchesAnyRegexp(regexMatching, text)
	}

	return keep
}

var regexMatching = []*common.RegexWithValue[synctypes.TransactionType]{
	{
		Regexp: regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) a (?P<place>.+) desde (?:cta|T\.CRED) \*(?P<account>\d{4})\.`),
		Value:  synctypes.Expense,
	},
	{
		Regexp: regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) en (?P<place>[^\.]+)\..+T\.Cred \*(?P<account>\d{4})\.`),
		Value:  synctypes.Expense,
	},
	{
		Regexp: regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) en (?P<place>.+)\..+T\.(?:Cred|Deb) \*(?P<account>\d{4})\.`),
		Value:  synctypes.Expense,
	},
	{
		Regexp: regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) por \$(?P<value>[0-9,\.]+) desde cta \*(?P<account>\d{4}).+cta (?P<place>\d{9,16})\.`),
		Value:  synctypes.Expense,
	},
	{
		Regexp: regexp.MustCompile(`Realizaste una (?P<type>\w+) con QR por \$(?P<value>[0-9,\.]+), desde cta \*(?P<account>\d{4}) a cta (?P<place>\d{9,16})\.`),
		Value:  synctypes.Expense,
	},
	{
		Regexp: regexp.MustCompile(`Bancolombia le informa (?P<type>\w+) de pago de (?P<place>[A-Z\s]+) por \$(?P<value>[0-9,\.]+) en su cuenta (?P<account>[A-Z\s]+)\s.+\.`),
		Value:  synctypes.Income,
	},
	{
		Regexp: regexp.MustCompile(`Bancolombia te informa (?P<type>\w+) transferencia de (?P<place>[A-Z\s]+) por \$(?P<value>[0-9,\.]+) en la cuenta \*(?P<account>[0-9]+)\.`),
		Value:  synctypes.Income,
	},
	{
		Regexp: regexp.MustCompile(`Bancolombia le informa un (?P<type>\w+) (?P<place>[\w\s]+) por \$(?P<value>[0-9,\.]+) en su Cuenta (?P<account>\w+)\.`),
		Value:  synctypes.Income,
	},
}

func (b Bancolombia) ExtractTransactionInfoFromMessage(msg imaptypes.Message) (*synctypes.TransactionInfo, error) {
	text := string(msg.RawBody)

	isFound, selectedRegexp := common.GenericMatchesAnyRegexp(regexMatching, text)
	if !isFound {
		return nil, fmt.Errorf("message did not match any regexp from Bancolombia")
	}

	// sanity check
	if !selectedRegexp.Value.IsValid() {
		return nil, fmt.Errorf("transaction type is not valid")
	}

	result := common.GenericExtractFieldsStringWithRegexp(text, selectedRegexp)

	if !common.ContainsAllRequiredFields(result) {
		return nil, fmt.Errorf("message does not contain all required fields - result [%+v]", result)
	}

	value, err := getValueFromText(result["value"])
	if err != nil {
		return nil, err
	}

	return &synctypes.TransactionInfo{
		Bank:            b,
		MsgId:           msg.SeqNum,
		TransactionType: selectedRegexp.Value,
		Type:            result["type"],
		Place:           result["place"],
		Value:           value,
		Account:         result["account"],
		Date:            msg.Envelope.Date,
	}, nil
}

// This would be way easier if Bancolombia had a consistent use of commas and dots inside the currency
var currencyRegexp = regexp.MustCompile(`^(?P<integer>[0-9\.,]+)[\.,](?P<decimal>\d{2})$`)
var currencyRegexpWithoutDecimal = regexp.MustCompile(`^(?P<integer>[0-9\.,]+)`)

func getValueFromTextWithDecimal(s string) (string, error) {
	if !currencyRegexp.MatchString(s) {
		return "", fmt.Errorf("string [%s] does not match regex [%s]", s, currencyRegexp.String())
	}

	res := common.ExtractFieldsStringWithRegexp(s, currencyRegexp)
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
		return "", fmt.Errorf("string [%s] does not match regex without decimal [%s]", s, currencyRegexp.String())
	}

	res := common.ExtractFieldsStringWithRegexp(s, currencyRegexpWithoutDecimal)
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

func getValueFromText(s string) (synctypes.Currency, error) {
	valueStr, err := getValueFromTextWithDecimal(s)
	if err != nil {
		valueStr, err = getValueFromTextWithoutDecimal(s)
	}

	if err != nil {
		return synctypes.Currency{}, err
	}

	value, err := strconv.ParseFloat(valueStr, 64)

	var currency synctypes.Currency
	currency.Code = "COP"
	currency.Rate = &value

	return currency, err
}
