package bancolombia

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/bank/banktypes"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types/currency"
)

const CopCode = "COP"

type transInfo struct {
	Place           string
	Account         string
	Value           currency.Amount
	TransactionType banktypes.TrxType
}

type result struct {
	Err    error
	Result transInfo
}

type regexpTest struct {
	Body   string
	Result result
}

var regexpExpressions []regexpTest = []regexpTest{
	{
		Body: "Bancolombia le informa compra por $30,000.00 a Prueba desde cta *0000.",
		Result: result{
			Result: transInfo{
				TransactionType: banktypes.Expense,
				Place:           "Prueba",
				Account:         "0000",
				Value:           generateCurrency(CopCode, 30000.0),
			},
		},
	},
	{
		Body: "Bancolombia le informa Compra por $3.150,00 en EXITO EXPRESS AVE 19 19:02. 20/09/2022 T.Deb *5021.",
		Result: result{
			Result: transInfo{
				TransactionType: banktypes.Expense,
				Place:           "EXITO EXPRESS AVE 19 19:02",
				Account:         "5021",
				Value:           generateCurrency(CopCode, 3150.0),
			},
		},
	},
	{
		Body: "Bancolombia le informa Retiro por $70.000,00 en CALLE100-2. Hora 16:44 01/09/2022 T.Deb *5021.",
		Result: result{
			Result: transInfo{
				TransactionType: banktypes.Expense,
				Place:           "CALLE100-2",
				Account:         "5021",
				Value:           generateCurrency(CopCode, 70000.0),
			},
		},
	},
	{
		Body: "Bancolombia le informa Retiro por $100.000,00 en EXITO_FUSAG. Hora 18:20 14/08/2022 T.Deb *5021.",
		Result: result{
			Result: transInfo{
				TransactionType: banktypes.Expense,
				Place:           "EXITO_FUSAG",
				Account:         "5021",
				Value:           generateCurrency(CopCode, 100000.0),
			},
		},
	},
	{
		Body: "Bancolombia le informa Retiro por $150.000,00 en MF_OCEANMA2. Hora 10:30 13/07/2022 T.Deb *5021.",
		Result: result{
			Result: transInfo{
				TransactionType: banktypes.Expense,
				Place:           "MF_OCEANMA2",
				Account:         "5021",
				Value:           generateCurrency(CopCode, 150000.0),
			},
		},
	},
	{
		Body: "Bancolombia te informa Pago por $112,400.00 a A Toda Hora SA desde producto *9785.",
		Result: result{
			Result: transInfo{
				TransactionType: banktypes.Expense,
				Place:           "A Toda Hora SA",
				Account:         "9785",
				Value:           generateCurrency(CopCode, 112400.0),
			},
		},
	},
	{
		Body: "Bancolombia le informa Compra por $23.050,00 en RAPPI RESTAURANTE 11:25. 04/01/2023 T.Cred *3616.",
		Result: result{
			Result: transInfo{
				TransactionType: banktypes.Expense,
				Place:           "RAPPI RESTAURANTE 11:25",
				Account:         "3616",
				Value:           generateCurrency(CopCode, 23050.0),
			},
		},
	},
}

func generateCurrency(code string, rate float64) currency.Amount {
	return currency.Amount{
		Number: rate,
		Code:   code,
	}
}

type testMessage struct {
	id      uint32
	from    []string
	to      []string
	subject string
	date    time.Time
	body    []byte
}

func (m testMessage) ID() uint32      { return m.id }
func (m testMessage) From() []string  { return m.from }
func (m testMessage) To() []string    { return m.to }
func (m testMessage) Subject() string { return m.subject }
func (m testMessage) Date() time.Time { return m.date }
func (m testMessage) Body() []byte    { return m.body }

func generateTestMessage(body string) banktypes.Message {
	return testMessage{
		id:      1,
		from:    []string{},
		to:      []string{},
		subject: "",
		date:    time.Now(),
		body:    []byte(body),
	}
}

func Test_RegexpExpressions(t *testing.T) {
	bank := Bancolombia{}

	for _, r := range regexpExpressions {
		msg := generateTestMessage(r.Body)

		res, err := bank.ExtractTransactionInfoFromMessage(msg)

		if r.Result.Err == nil {
			assert.NotNil(t, res)
			assert.Equal(t, res.Type, r.Result.Result.TransactionType)
			assert.Equal(t, res.Description, r.Result.Result.Place)
			assert.Equal(t, res.Account, r.Result.Result.Account)
			assert.Equal(t, res.Value, r.Result.Result.Value)
		} else {
			assert.Error(t, err)
		}
	}
}
